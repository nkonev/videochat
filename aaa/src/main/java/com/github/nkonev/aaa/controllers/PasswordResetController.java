package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.repository.redis.PasswordResetTokenRepository;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import com.github.nkonev.aaa.exception.PasswordResetTokenNotFoundException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.services.EmailService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;
import javax.validation.Valid;
import javax.validation.constraints.NotEmpty;
import javax.validation.constraints.NotNull;
import javax.validation.constraints.Size;
import java.time.Duration;
import java.util.Optional;
import java.util.UUID;

@RestController
@Transactional
public class PasswordResetController {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordResetTokenRepository passwordResetTokenRepository;

    @Value("${custom.password-reset.token.ttl-minutes}")
    private long passwordResetTokenTtlMinutes;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private EmailService emailService;

    private static final Logger LOGGER = LoggerFactory.getLogger(PasswordResetController.class);

    /**
     * https://www.owasp.org/index.php/Forgot_Password_Cheat_Sheet
     * https://stackoverflow.com/questions/1102781/best-way-for-a-forgot-password-implementation/1102817#1102817
     * Yes, if your email is stolen you can lost your account
     * @param email
     */
    @PostMapping(value = Constants.Urls.API+ Constants.Urls.REQUEST_PASSWORD_RESET)
    public void requestPasswordReset(String email) {
        String uuid = UUID.randomUUID().toString();

        Optional<UserAccount> userAccountOptional = userAccountRepository.findByEmail(email);
        if (!userAccountOptional.isPresent()) {
            LOGGER.warn("Skipping sent request password reset email '{}' because this email is not found", email);
            return; // we care for for email leak
        }
        UserAccount userAccount = userAccountOptional.get();

        Duration ttl = Duration.ofMinutes(passwordResetTokenTtlMinutes);

        PasswordResetToken passwordResetToken = new PasswordResetToken(uuid, userAccount.id(), ttl.getSeconds());

        passwordResetToken = passwordResetTokenRepository.save(passwordResetToken);

        emailService.sendPasswordResetToken(userAccount.email(), passwordResetToken, userAccount.username());
    }

    record PasswordResetDto(
        @NotNull
        UUID passwordResetToken,

        @Size(min = Constants.MIN_PASSWORD_LENGTH, max = Constants.MAX_PASSWORD_LENGTH)
        @NotEmpty
        String newPassword
    ) {}

    @PostMapping(value = Constants.Urls.API + Constants.Urls.PASSWORD_RESET_SET_NEW)
    public void resetPassword(@RequestBody @Valid PasswordResetDto passwordResetDto) {

        // webpage parses token uuid from URL
        // .. and js sends this request

        Optional<PasswordResetToken> passwordResetTokenOptional = passwordResetTokenRepository.findById(passwordResetDto.passwordResetToken());
        if (!passwordResetTokenOptional.isPresent()) {
            throw new PasswordResetTokenNotFoundException("password reset token not found or expired");
        }
        PasswordResetToken passwordResetToken = passwordResetTokenOptional.get();
        Optional<UserAccount> userAccountOptional = userAccountRepository.findById(passwordResetToken.userId());
        if(!userAccountOptional.isPresent()) {
            return;
        }

        UserAccount userAccount = userAccountOptional.get();

        userAccount = userAccount.withPassword(passwordEncoder.encode(passwordResetDto.newPassword()));
        userAccount = userAccountRepository.save(userAccount);
        return;
    }
}
