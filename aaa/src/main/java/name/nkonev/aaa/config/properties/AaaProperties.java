package name.nkonev.aaa.config.properties;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.time.Duration;

@ConfigurationProperties(prefix = "custom")
public record AaaProperties(
    String apiUrl,
    String frontendUrl,
    String registrationConfirmExitTokenNotFoundUrl,
    String registrationConfirmExitUserNotFoundUrl,
    String registrationConfirmExitSuccessUrl,
    String passwordResetEnterNewUrl,
    String confirmChangeEmailExitSuccessUrl,
    String confirmChangeEmailExitTokenNotFoundUrl,

    HttpClientProperties httpClient,
    LdapProperties ldap,
    EmailProperties email,
    ConfirmationProperties confirmation,
    PasswordResetProperties passwordReset,
    Duration onlineEstimation,
    SchedulersProperties schedulers,
    CsrfProperties csrf,

    boolean debugResponse
) { }
