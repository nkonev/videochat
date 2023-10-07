package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_PASSWORD_RESET_EMAILS_NAME;
import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_USER_CONFIRMATION_EMAILS_NAME;


@Service
public class AsyncEmailService {
    @Autowired
    private EmailService emailService;

    @Autowired
    private RabbitTemplate rabbitTemplate;

    record UserConfirmationEmailDTO(
        String email,
        UserConfirmationToken userConfirmationToken,
        String login
    ) {}

    record PasswordResetEmailDTO(
        String email,
        PasswordResetToken passwordResetToken,
        String login
    ) {}

    public void sendUserConfirmationToken(String email, UserConfirmationToken userConfirmationToken, String login) {
        rabbitTemplate.convertAndSend(QUEUE_USER_CONFIRMATION_EMAILS_NAME, new UserConfirmationEmailDTO(email, userConfirmationToken, login));
    }

    public void sendPasswordResetToken(String email, PasswordResetToken passwordResetToken, String login) {
        rabbitTemplate.convertAndSend(QUEUE_PASSWORD_RESET_EMAILS_NAME, new PasswordResetEmailDTO(email, passwordResetToken, login));
    }

    @RabbitListener(queues = QUEUE_USER_CONFIRMATION_EMAILS_NAME)
    public void handleUserConfirmation(UserConfirmationEmailDTO dto) {
        emailService.sendUserConfirmationToken(dto.email(), dto.userConfirmationToken(), dto.login());
    }

    @RabbitListener(queues = QUEUE_PASSWORD_RESET_EMAILS_NAME)
    public void handlePasswordReset(PasswordResetEmailDTO dto) {
        emailService.sendPasswordResetToken(dto.email(), dto.passwordResetToken(), dto.login());
    }
}
