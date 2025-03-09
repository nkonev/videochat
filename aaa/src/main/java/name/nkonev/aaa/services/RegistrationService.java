package name.nkonev.aaa.services;

import jakarta.servlet.http.HttpServletRequest;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.EditUserDTO;
import name.nkonev.aaa.dto.Language;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.redis.UserConfirmationToken;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.repository.jdbc.UserSettingsRepository;
import name.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import name.nkonev.aaa.security.LoginListener;
import name.nkonev.aaa.security.SecurityUtils;
import jakarta.servlet.http.HttpSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

import java.util.Optional;
import java.util.UUID;

import static name.nkonev.aaa.converter.UserAccountConverter.validateLengthAndTrimLogin;
import static name.nkonev.aaa.converter.UserAccountConverter.validateLengthEmail;

@Service
public class RegistrationService {
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private UserConfirmationTokenRepository userConfirmationTokenRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @Autowired
    private CheckService userService;

    @Autowired
    private LoginListener loginListener;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private RefererService refererService;

    @Autowired
    private UserSettingsRepository userSettingsRepository;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationService.class);

    public void register(EditUserDTO editUserDTO, Language language, String referer, HttpServletRequest httpServletRequest) {
        var userAccountEditDTO = UserAccountConverter.normalize(editUserDTO, false);
        var login = validateLengthAndTrimLogin(userAccountEditDTO.login(), false);
        var userAccountDTO = userAccountEditDTO.withLogin(login);
        validateLengthEmail(userAccountDTO.email());

        var userAccountOuter = transactionTemplate.execute(status -> {
            userService.checkLoginIsFreeOrThrow(userAccountDTO.login());

            if (!userService.checkEmailIsFree(userAccountDTO.email())){
                return null; // we care for user email leak
            }

            UserAccount userAccount = userAccountConverter.buildUserAccountEntityForInsert(userAccountDTO);

            userAccount = userAccountRepository.save(userAccount);
            userSettingsRepository.insertDefault(userAccount.id());
            userSettingsRepository.updateLanguage(userAccount.id(), language);
            UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount, referer, httpServletRequest);
            asyncEmailService.sendUserConfirmationToken(userAccount.email(), userConfirmationToken, userAccount.login(), language);
            return userAccount;
        });
        if (userAccountOuter != null) {
            eventService.notifyProfileCreated(userAccountOuter);
        }
    }

    /**
     * Handles confirmations.
     * In frontend router also should implement follows pages
     * /confirm -- success confirmation
     * /confirm/registration/token-not-found
     * /confirm/registration/user-not-found
     * @param uuid
     * @return
     */
    @Transactional
    public String confirm(UUID uuid, HttpSession httpSession, HttpServletRequest httpServletRequest) {
        Optional<UserConfirmationToken> userConfirmationTokenOptional = userConfirmationTokenRepository.findById(uuid);
        if (!userConfirmationTokenOptional.isPresent()) {
            LOGGER.info("For uuid {}, confirm user token is not found", uuid);
            return aaaProperties.registrationConfirmExitTokenNotFoundUrl();
        }
        UserConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        Optional<UserAccount> userAccountOptional = userAccountRepository.findById(userConfirmationToken.userId());
        if (!userAccountOptional.isPresent()) {
            LOGGER.info("For uuid {}, user account is not found", uuid);
            return aaaProperties.registrationConfirmExitUserNotFoundUrl();
        }
        UserAccount userAccount = userAccountOptional.get();
        if (userAccount.confirmed()) {
            LOGGER.warn("Somebody attempts secondary confirm already confirmed user account with email='{}'", userAccount);
            return aaaProperties.registrationConfirmExitSuccessUrl();
        }

        userAccount = userAccount.withConfirmed(true);
        userAccount = userAccountRepository.save(userAccount);

        userConfirmationTokenRepository.deleteById(uuid);

        var auth = userAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityUtils.setToContext(httpSession, auth);
        loginListener.onApplicationEvent(auth);
        eventService.notifyProfileUpdated(userAccount);

        var referer = userConfirmationToken.referer();
        if (StringUtils.hasLength(referer)) {
            LOGGER.info("Redirecting user with id {} with addr {} to the restored referer url {}", SecurityUtils.getPrincipal().getId(), httpServletRequest.getHeader("x-real-ip"), referer);
            return referer;
        } else {
            return aaaProperties.registrationConfirmExitSuccessUrl();
        }
    }

    @Transactional
    public void resendConfirmationToken(String email, Language language, String referer, HttpServletRequest httpServletRequest) {
        Optional<UserAccount> userAccountOptional = userAccountRepository.findByEmail(email);
        if(!userAccountOptional.isPresent()){
            LOGGER.warn("Skipping sent subsequent confirmation email '{}' because this email is not found", email);
            return; // we care for for email leak
        }
        UserAccount userAccount = userAccountOptional.get();
        if (userAccount.confirmed()) {
            // this account already confirmed
            LOGGER.warn("Skipping sent subsequent confirmation email '{}' because this user account already confirmed", email);
            return; // we care for for email leak
        }

        UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount, referer, httpServletRequest);
        asyncEmailService.sendUserConfirmationToken(email, userConfirmationToken, userAccount.login(), language);
    }

    private UserConfirmationToken createUserConfirmationToken(UserAccount userAccount, String referer, HttpServletRequest currentHttpRequest) {
        Assert.isTrue(!userAccount.confirmed(), "user account mustn't be confirmed");

        long seconds = aaaProperties.confirmation().registration().token().ttl().getSeconds(); // Redis requires seconds
        var validReferer = refererService.getRefererOrEmpty(referer);
        if (StringUtils.hasLength(validReferer)) {
            LOGGER.info("Storing referer url {} for still non-user with addr {}", validReferer, currentHttpRequest.getHeader("x-real-ip"));
        }

        UUID tokenUuid = UUID.randomUUID();
        UserConfirmationToken userConfirmationToken = new UserConfirmationToken(tokenUuid, userAccount.id(), validReferer, seconds);
        return userConfirmationTokenRepository.save(userConfirmationToken);
    }
}
