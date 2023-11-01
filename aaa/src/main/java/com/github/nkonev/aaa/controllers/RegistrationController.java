package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import com.github.nkonev.aaa.security.SecurityUtils;
import com.github.nkonev.aaa.services.AsyncEmailService;
import com.github.nkonev.aaa.services.UserService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Controller;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;
import org.springframework.web.bind.annotation.*;

import javax.validation.Valid;
import java.time.Duration;
import java.util.Optional;
import java.util.UUID;

@Controller
@Transactional
public class RegistrationController {
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private UserConfirmationTokenRepository userConfirmationTokenRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @Value("${custom.confirmation.registration.token.ttl-minutes}")
    private long userConfirmationTokenTtlMinutes;

    @Autowired
    private UserService userService;

    @Autowired
    private CustomConfig customConfig;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationController.class);

    private UserConfirmationToken createUserConfirmationToken(UserAccount userAccount) {
        Assert.isTrue(!userAccount.enabled(), "user account mustn't be enabled");

        Duration ttl = Duration.ofMinutes(userConfirmationTokenTtlMinutes);
        long seconds = ttl.getSeconds(); // Redis requires seconds

        UUID tokenUuid = UUID.randomUUID();
        UserConfirmationToken userConfirmationToken = new UserConfirmationToken(tokenUuid.toString(), userAccount.id(), seconds);
        return userConfirmationTokenRepository.save(userConfirmationToken);
    }

    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
    @ResponseBody
    public void register(@RequestBody @Valid EditUserDTO userAccountDTO) {
        userAccountDTO = UserAccountConverter.trimAndValidateNonAouth2Login(userAccountDTO);

        userService.checkLoginIsFree(userAccountDTO);
        if(!userService.checkEmailIsFree(userAccountDTO)){
            return; // we care for user email leak
        }

        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForInsert(userAccountDTO, passwordEncoder);

        userAccount = userAccountRepository.save(userAccount);
        UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount);
        asyncEmailService.sendUserConfirmationToken(userAccount.email(), userConfirmationToken, userAccount.username());
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
    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER_CONFIRM)
    public String confirm(@RequestParam(Constants.Urls.UUID) UUID uuid) {
        String stringUuid = uuid.toString();
        Optional<UserConfirmationToken> userConfirmationTokenOptional = userConfirmationTokenRepository.findById(stringUuid);
        if (!userConfirmationTokenOptional.isPresent()) {
            return "redirect:" + customConfig.getRegistrationConfirmExitTokenNotFoundUrl();
        }
        UserConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        Optional<UserAccount> userAccountOptional = userAccountRepository.findById(userConfirmationToken.userId());
        if (!userAccountOptional.isPresent()) {
            return "redirect:" + customConfig.getRegistrationConfirmExitUserNotFoundUrl();
        }
        UserAccount userAccount = userAccountOptional.get();
        if (userAccount.enabled()) {
            LOGGER.warn("Somebody attempts secondary confirm already confirmed user account with email='{}'", userAccount);
            return "redirect:" + customConfig.getRegistrationConfirmExitSuccessUrl();
        }

        userAccount = userAccount.withEnabled(true);
        userAccount = userAccountRepository.save(userAccount);

        userConfirmationTokenRepository.deleteById(stringUuid);
        SecurityUtils.authenticate(userAccount);

        return "redirect:" + customConfig.getRegistrationConfirmExitSuccessUrl();
    }

    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL)
    @ResponseBody
    public void resendConfirmationToken(@RequestParam(value = "email") String email) {
        Optional<UserAccount> userAccountOptional = userAccountRepository.findByEmail(email);
        if(!userAccountOptional.isPresent()){
            LOGGER.warn("Skipping sent subsequent confirmation email '{}' because this email is not found", email);
            return; // we care for for email leak
        }
        UserAccount userAccount = userAccountOptional.get();
        if (userAccount.enabled()) {
            // this account already confirmed
            LOGGER.warn("Skipping sent subsequent confirmation email '{}' because this user account already enabled", email);
            return; // we care for for email leak
        }

        UserConfirmationToken userConfirmationToken = createUserConfirmationToken(userAccount);
        asyncEmailService.sendUserConfirmationToken(email, userConfirmationToken, userAccount.username());
    }

}
