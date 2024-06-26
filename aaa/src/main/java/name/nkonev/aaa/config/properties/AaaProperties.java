package name.nkonev.aaa.config.properties;

import org.springframework.boot.context.properties.ConfigurationProperties;
import name.nkonev.aaa.utils.UrlUtils;
import java.time.Duration;
import java.util.List;

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

    boolean debugResponse,

    String allowedAvatarUrls
) {
    public List<String> getAllowedAvatarUrlsList() {
        if (allowedAvatarUrls == null) {
            return List.of();
        }
        if (allowedAvatarUrls.isEmpty()) {
            return List.of("");
        }
        return List.of(allowedAvatarUrls.split(","));
    }
}

