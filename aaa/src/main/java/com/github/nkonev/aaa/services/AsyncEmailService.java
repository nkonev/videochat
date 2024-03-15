package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import static com.github.nkonev.aaa.config.RabbitMqConfig.*;


@Service
public class AsyncEmailService {
    @Autowired
    private EmailService emailService;

    @Autowired
    private RabbitTemplate rabbitTemplate;

    record UserConfirmationEmailDTO(
        String email,
        UserConfirmationToken userConfirmationToken,
        String login,
        Language language
    ) {}

    record PasswordResetEmailDTO(
        String email,
        PasswordResetToken passwordResetToken,
        String login,
        Language language
    ) {}

    record ChangeEmailConfirmationDTO(
        String newEmail,
        ChangeEmailConfirmationToken changeEmailConfirmationToken,
        String login,
        Language language
    ) {}

    public record ArbitraryEmailDto(
        String recipient,
        String subject,
        String body
    ){}

    public void sendUserConfirmationToken(String email, UserConfirmationToken userConfirmationToken, String login, Language language) {
        rabbitTemplate.convertAndSend(QUEUE_USER_CONFIRMATION_EMAILS_NAME, new UserConfirmationEmailDTO(email, userConfirmationToken, login, language));
    }

    public void sendPasswordResetToken(String email, PasswordResetToken passwordResetToken, String login, Language language) {
        rabbitTemplate.convertAndSend(QUEUE_PASSWORD_RESET_EMAILS_NAME, new PasswordResetEmailDTO(email, passwordResetToken, login, language));
    }

    public void sendChangeEmailConfirmationToken(String newEmail, ChangeEmailConfirmationToken changeEmailConfirmationToken, String login, Language language) {
        rabbitTemplate.convertAndSend(QUEUE_CHANGE_EMAIL_CONFIRMATION_NAME, new ChangeEmailConfirmationDTO(newEmail, changeEmailConfirmationToken, login, language));
    }

    public void sendArbitraryEmail(ArbitraryEmailDto dto) {
        rabbitTemplate.convertAndSend(QUEUE_ARBITRARY_EMAILS_NAME, dto);
    }

    @RabbitListener(queues = QUEUE_USER_CONFIRMATION_EMAILS_NAME)
    public void handleUserConfirmation(UserConfirmationEmailDTO dto) {
        emailService.sendUserConfirmationToken(dto.email(), dto.userConfirmationToken(), dto.login(), dto.language());
    }

    @RabbitListener(queues = QUEUE_PASSWORD_RESET_EMAILS_NAME)
    public void handlePasswordReset(PasswordResetEmailDTO dto) {
        emailService.sendPasswordResetToken(dto.email(), dto.passwordResetToken(), dto.login(), dto.language());
    }

    @RabbitListener(queues = QUEUE_CHANGE_EMAIL_CONFIRMATION_NAME)
    public void handleChangeEmailConfirmationToken(ChangeEmailConfirmationDTO dto) {
        emailService.changeEmailConfirmationToken(dto.newEmail(), dto.changeEmailConfirmationToken(), dto.login(), dto.language());
    }

    @RabbitListener(queues = QUEUE_ARBITRARY_EMAILS_NAME)
    public void handleArbitraryEmail(ArbitraryEmailDto dto) {
        emailService.sendArbitraryEmail(dto.recipient(), dto.subject(), dto.body());
    }
}
