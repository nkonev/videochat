package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import freemarker.template.Configuration;
import freemarker.template.Template;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.io.StringWriter;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.util.Map;

/**
 * Performs sending entity to email
 */
@Service
public class EmailService {

    @Autowired
    private JavaMailSender mailSender;

    @Autowired
    private CustomConfig customConfig;

    @Value("${custom.email.from}")
    private String from;

    @Value("${custom.registration.email.subject}")
    private String registrationSubject;

    @Value("${custom.password-reset.email.subject}")
    private String passwordResetSubject;

    @Autowired
    private Configuration freemarkerConfiguration;

    private static final String REG_LINK_PLACEHOLDER = "REGISTRATION_LINK_PLACEHOLDER";
    private static final String PASSWORD_RESET_LINK_PLACEHOLDER = "PASSWORD_RESET_LINK_PLACEHOLDER";
    private static final String LOGIN_PLACEHOLDER = "LOGIN";

    public void sendUserConfirmationToken(String email, UserConfirmationToken userConfirmationToken, String login) {
        // https://yandex.ru/support/mail-new/mail-clients.html
        // https://docs.spring.io/spring-boot/docs/current/reference/html/boot-features-email.html
        // http://docs.spring.io/spring/docs/4.3.10.RELEASE/spring-framework-reference/htmlsingle/#mail-usage-simple
        SimpleMailMessage msg = new SimpleMailMessage();
        msg.setFrom(from);
        msg.setSubject(registrationSubject);
        msg.setTo(email);

        final var regLink = customConfig.getBaseUrl() + Constants.Urls.CONFIRM+ "?"+ Constants.Urls.UUID +"=" + userConfirmationToken.uuid() + "&login=" + URLEncoder.encode(login, StandardCharsets.UTF_8);
        final var text = renderTemplate("confirm-registration.ftlh",
                Map.of(REG_LINK_PLACEHOLDER, regLink, LOGIN_PLACEHOLDER, login));

        msg.setText(text);
        mailSender.send(msg);
    }

    public void sendPasswordResetToken(String email, PasswordResetToken passwordResetToken, String login) {
        SimpleMailMessage msg = new SimpleMailMessage();
        msg.setFrom(from);
        msg.setSubject(passwordResetSubject);
        msg.setTo(email);

        final var passwordResetLink = customConfig.getBaseUrl() + Constants.Urls.PASSWORD_RESET+ "?"+ Constants.Urls.UUID +"=" + passwordResetToken.uuid() + "&login=" + URLEncoder.encode(login, StandardCharsets.UTF_8);
        final var text = renderTemplate("password-reset.ftlh",
                Map.of(PASSWORD_RESET_LINK_PLACEHOLDER, passwordResetLink, LOGIN_PLACEHOLDER, login));

        msg.setText(text);

        mailSender.send(msg);
    }

    private String renderTemplate(String templateNameWithExtension, Object model) {
        try {
            final Template template = freemarkerConfiguration.getTemplate(templateNameWithExtension);
            final var stringWriter = new StringWriter();
            template.process(model, stringWriter);
            return stringWriter.toString();
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }
}
