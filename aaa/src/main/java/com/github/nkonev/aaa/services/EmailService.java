package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.entity.redis.PasswordResetToken;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import freemarker.template.Configuration;
import freemarker.template.Template;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.mail.javamail.MimeMessageHelper;
import org.springframework.stereotype.Service;

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

    @Autowired
    private Configuration freemarkerConfiguration;

    private static final String REG_LINK_PLACEHOLDER = "REGISTRATION_LINK_PLACEHOLDER";
    private static final String PASSWORD_RESET_LINK_PLACEHOLDER = "PASSWORD_RESET_LINK_PLACEHOLDER";
    private static final String LOGIN_PLACEHOLDER = "LOGIN";

    private static final Logger LOGGER = LoggerFactory.getLogger(EmailService.class);

    public void sendUserConfirmationToken(String recipient, UserConfirmationToken userConfirmationToken, String login, Language language) {
        // https://yandex.ru/support/mail-new/mail-clients.html
        // https://docs.spring.io/spring-boot/docs/current/reference/html/boot-features-email.html
        // http://docs.spring.io/spring/docs/4.3.10.RELEASE/spring-framework-reference/htmlsingle/#mail-usage-simple
        try {
            var mimeMessage = mailSender.createMimeMessage();
            var helper = new MimeMessageHelper(mimeMessage, "UTF-8");

            helper.setFrom(from);
            final var subj = renderTemplate("confirm_registration_subject_%s.ftlh".formatted(language), Map.of());
            helper.setSubject(subj);
            helper.setTo(recipient);

            final var regLink = customConfig.getBaseUrl() + Constants.Urls.REGISTER_CONFIRM + "?" + Constants.Urls.UUID + "=" + userConfirmationToken.uuid();
            final var text = renderTemplate("confirm_registration_body_%s.ftlh".formatted(language),
                Map.of(REG_LINK_PLACEHOLDER, regLink, LOGIN_PLACEHOLDER, login));

            LOGGER.trace("For registration confirmation '{}' generated email text '{}'", recipient, text);
            helper.setText(text, true);

            mailSender.send(mimeMessage);
        } catch (Exception e) {
            throw new RuntimeException("Unable to send confirmation token", e);
        }
    }

    public void sendPasswordResetToken(String recipient, PasswordResetToken passwordResetToken, String login, Language language) {
        try {
            var mimeMessage = mailSender.createMimeMessage();
            var helper = new MimeMessageHelper(mimeMessage, "UTF-8");

            helper.setFrom(from);
            final var subj = renderTemplate("password_reset_subject_%s.ftlh".formatted(language), Map.of());
            helper.setSubject(subj);
            helper.setTo(recipient);

            final var passwordResetLink = customConfig.getPasswordRestoreEnterNew() + "?"+ Constants.Urls.UUID +"=" + passwordResetToken.uuid() + "&login=" + URLEncoder.encode(login, StandardCharsets.UTF_8);
            final var text = renderTemplate("password_reset_body_%s.ftlh".formatted(language),
                    Map.of(PASSWORD_RESET_LINK_PLACEHOLDER, passwordResetLink, LOGIN_PLACEHOLDER, login));
            LOGGER.trace("For password reset '{}' generated email text '{}'", recipient, text);
            helper.setText(text, true);

            mailSender.send(mimeMessage);
        } catch (Exception e) {
            throw new RuntimeException("Unable to send confirmation token", e);
        }
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

    public void sendArbitraryEmail(String recipient, String subject, String body) {
        SimpleMailMessage msg = new SimpleMailMessage();
        msg.setFrom(from);
        msg.setSubject(subject);
        msg.setTo(recipient);
        msg.setText(body);

        mailSender.send(msg);
    }
}
