package name.nkonev.aaa.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "custom")
public record AaaProperties(
    String baseUrl,
    String registrationConfirmExitTokenNotFoundUrl,
    String registrationConfirmExitUserNotFoundUrl,
    String registrationConfirmExitSuccessUrl,
    String passwordResetEnterNewUrl,
    String confirmChangeEmailExitSuccessUrl,
    String confirmChangeEmailExitTokenNotFoundUrl
) { }
