package name.nkonev.aaa.services;

import name.nkonev.aaa.config.CustomConfig;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.EditUserDTO;
import name.nkonev.aaa.dto.Language;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.redis.UserConfirmationToken;
import name.nkonev.aaa.exception.BadRequestException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import name.nkonev.aaa.security.LoginListener;
import name.nkonev.aaa.security.SecurityUtils;
import jakarta.servlet.http.HttpSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.Assert;

import java.time.Duration;
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

    @Value("${custom.confirmation.registration.token.ttl}")
    private Duration userConfirmationTokenTtl;

    @Autowired
    private CheckService userService;

    @Autowired
    private CustomConfig customConfig;

    @Autowired
    private LoginListener loginListener;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationService.class);

    public void register(EditUserDTO editUserDTO, Language language) {
        var userAccountEditDTO = UserAccountConverter.normalize(editUserDTO, false);
        var login = validateLengthAndTrimLogin(userAccountEditDTO.login(), false);
        var userAccountDTO = userAccountEditDTO.withLogin(login);
        validateLengthEmail(userAccountDTO.email());

        var userAccountOuter = transactionTemplate.execute(status -> {
            userService.checkLoginIsFree(userAccountDTO.login());

            if (!userService.checkEmailIsFree(userAccountDTO.email())){
                return null; // we care for user email leak
            }

            UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForInsert(userAccountDTO, passwordEncoder);

            userAccount = userAccountRepository.save(userAccount);
            UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount);
            asyncEmailService.sendUserConfirmationToken(userAccount.email(), userConfirmationToken, userAccount.username(), language);
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
    public String confirm(UUID uuid, HttpSession httpSession) {
        Optional<UserConfirmationToken> userConfirmationTokenOptional = userConfirmationTokenRepository.findById(uuid);
        if (!userConfirmationTokenOptional.isPresent()) {
            return customConfig.getRegistrationConfirmExitTokenNotFoundUrl();
        }
        UserConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        Optional<UserAccount> userAccountOptional = userAccountRepository.findById(userConfirmationToken.userId());
        if (!userAccountOptional.isPresent()) {
            return customConfig.getRegistrationConfirmExitUserNotFoundUrl();
        }
        UserAccount userAccount = userAccountOptional.get();
        if (userAccount.confirmed()) {
            LOGGER.warn("Somebody attempts secondary confirm already confirmed user account with email='{}'", userAccount);
            return customConfig.getRegistrationConfirmExitSuccessUrl();
        }

        userAccount = userAccount.withConfirmed(true);
        userAccount = userAccountRepository.save(userAccount);

        userConfirmationTokenRepository.deleteById(uuid);

        var auth = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityUtils.setToContext(httpSession, auth);
        loginListener.onApplicationEvent(auth);
        eventService.notifyProfileUpdated(userAccount);

        return customConfig.getRegistrationConfirmExitSuccessUrl();
    }

    @Transactional
    public void resendConfirmationToken(String email, Language language) {
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

        UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount);
        asyncEmailService.sendUserConfirmationToken(email, userConfirmationToken, userAccount.username(), language);
    }

    private UserConfirmationToken createUserConfirmationToken(UserAccount userAccount) {
        Assert.isTrue(!userAccount.confirmed(), "user account mustn't be confirmed");

        long seconds = userConfirmationTokenTtl.getSeconds(); // Redis requires seconds

        UUID tokenUuid = UUID.randomUUID();
        UserConfirmationToken userConfirmationToken = new UserConfirmationToken(tokenUuid, userAccount.id(), seconds);
        return userConfirmationTokenRepository.save(userConfirmationToken);
    }
}
