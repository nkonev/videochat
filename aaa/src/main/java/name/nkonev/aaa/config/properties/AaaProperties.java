package name.nkonev.aaa.config.properties;

import org.springframework.boot.context.properties.ConfigurationProperties;
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
    KeycloakProperties keycloak,
    VkontakteProperties vkontakte,
    GoogleProperties google,
    FacebookProperties facebook,
    EmailProperties email,
    ConfirmationProperties confirmation,
    PasswordResetProperties passwordReset,
    Duration onlineEstimation,
    SchedulersProperties schedulers,
    CsrfProperties csrf,

    boolean debugResponse,

    String allowedAvatarUrls,
    RoleMappings roleMappings,
    AdminsCornerProperties adminsCorner,

    Duration frontendSessionPingInterval
) {
    public List<String> getAllowedAvatarUrlsList() {
        if (allowedAvatarUrls == null) {
            return List.of();
        }
        return List.of(allowedAvatarUrls.split(","));
    }
}

