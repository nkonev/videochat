package name.nkonev.aaa.services;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.Language;
import name.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import name.nkonev.aaa.entity.redis.PasswordResetToken;
import name.nkonev.aaa.entity.redis.UserConfirmationToken;
import freemarker.template.Configuration;
import freemarker.template.Template;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
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
    private Configuration freemarkerConfiguration;

    @Autowired
    private AaaProperties aaaProperties;

    private static final String REG_LINK_PLACEHOLDER = "REGISTRATION_LINK_PLACEHOLDER";
    private static final String PASSWORD_RESET_LINK_PLACEHOLDER = "PASSWORD_RESET_LINK_PLACEHOLDER";

    private static final String CHANGE_EMAIL_CONFIRMATION_LINK_PLACEHOLDER = "CHANGE_EMAIL_CONFIRMATION_LINK_PLACEHOLDER";
    private static final String LOGIN_PLACEHOLDER = "LOGIN";

    private static final Logger LOGGER = LoggerFactory.getLogger(EmailService.class);

    public void sendUserConfirmationToken(String recipient, UserConfirmationToken userConfirmationToken, String login, Language language) {
        // https://yandex.ru/support/mail-new/mail-clients.html
        // https://docs.spring.io/spring-boot/docs/current/reference/html/boot-features-email.html
        // http://docs.spring.io/spring/docs/4.3.10.RELEASE/spring-framework-reference/htmlsingle/#mail-usage-simple
        try {
            var mimeMessage = mailSender.createMimeMessage();
            var helper = new MimeMessageHelper(mimeMessage, "UTF-8");

            helper.setFrom(aaaProperties.email().from());
            final var subj = renderTemplate("confirm_registration_subject_%s.ftlh".formatted(language), Map.of());
            helper.setSubject(subj);
            helper.setTo(recipient);

            final var regLink = aaaProperties.apiUrl() + Constants.Urls.REGISTER_CONFIRM + "?" + Constants.Urls.UUID + "=" + userConfirmationToken.uuid();
            final var text = renderTemplate("confirm_registration_body_%s.ftlh".formatted(language),
                Map.of(REG_LINK_PLACEHOLDER, regLink, LOGIN_PLACEHOLDER, login));

            LOGGER.trace("For registration confirmation of '{}' generated email text '{}'", recipient, text);
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

            helper.setFrom(aaaProperties.email().from());
            final var subj = renderTemplate("password_reset_subject_%s.ftlh".formatted(language), Map.of());
            helper.setSubject(subj);
            helper.setTo(recipient);

            final var passwordResetLink = aaaProperties.passwordResetEnterNewUrl() + "?"+ Constants.Urls.UUID +"=" + passwordResetToken.uuid() + "&login=" + URLEncoder.encode(login, StandardCharsets.UTF_8);
            final var text = renderTemplate("password_reset_body_%s.ftlh".formatted(language),
                    Map.of(PASSWORD_RESET_LINK_PLACEHOLDER, passwordResetLink, LOGIN_PLACEHOLDER, login));
            LOGGER.trace("For password reset of '{}' generated email text '{}'", recipient, text);
            helper.setText(text, true);

            mailSender.send(mimeMessage);
        } catch (Exception e) {
            throw new RuntimeException("Unable to send confirmation token", e);
        }
    }

    public void changeEmailConfirmationToken(ChangeEmailConfirmationToken changeEmailConfirmationToken, String login, Language language) {
        try {
            var mimeMessage = mailSender.createMimeMessage();
            var helper = new MimeMessageHelper(mimeMessage, "UTF-8");

            helper.setFrom(aaaProperties.email().from());
            final var subj = renderTemplate("confirm_change_email_subject_%s.ftlh".formatted(language), Map.of());
            helper.setSubject(subj);
            helper.setTo(changeEmailConfirmationToken.newEmail());

            final var confirmLink = aaaProperties.apiUrl() + Constants.Urls.CHANGE_EMAIL_CONFIRM + "?" + Constants.Urls.UUID + "=" + changeEmailConfirmationToken.uuid();
            final var text = renderTemplate("confirm_change_email_body_%s.ftlh".formatted(language),
                Map.of(CHANGE_EMAIL_CONFIRMATION_LINK_PLACEHOLDER, confirmLink, LOGIN_PLACEHOLDER, login));
            LOGGER.trace("For changing email to '{}' generated email text '{}'", changeEmailConfirmationToken.newEmail(), text);
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
        msg.setFrom(aaaProperties.email().from());
        msg.setSubject(subject);
        msg.setTo(recipient);
        msg.setText(body);

        mailSender.send(msg);
    }
}
