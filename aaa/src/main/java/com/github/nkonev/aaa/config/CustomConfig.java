package com.github.nkonev.aaa.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

import java.time.Duration;

@Configuration
public class CustomConfig {
    @Value("${custom.base-url}")
    private String baseUrl;

    @Value("${custom.registration-confirm-exit-token-not-found-url}")
    private String registrationConfirmExitTokenNotFoundUrl;

    @Value("${custom.registration-confirm-exit-user-not-found-url}")
    private String registrationConfirmExitUserNotFoundUrl;

    @Value("${custom.registration-confirm-exit-success-url}")
    private String registrationConfirmExitSuccessUrl;

    @Value("${custom.password-reset-enter-new-url}")
    private String passwordRestoreEnterNew;

    @Value("${custom.confirm-change-email-exit-success-url}")
    private String confirmChangeEmailExitSuccessUrl;

    @Value("${custom.confirm-change-email-exit-token-not-found-url}")
    private String confirmChangeEmailExitTokenNotFoundUrl;

    @Value("${http.client.connect-timeout:3s}")
    private Duration restClientConnectTimeout;

    @Value("${http.client.read-timeout:30s}")
    private Duration restClientReadTimeout;

    public String getBaseUrl() {
        return baseUrl;
    }

    public Duration getRestClientConnectTimeout() {
        return restClientConnectTimeout;
    }

    public Duration getRestClientReadTimeout() {
        return restClientReadTimeout;
    }

    public String getRegistrationConfirmExitTokenNotFoundUrl() {
        return registrationConfirmExitTokenNotFoundUrl;
    }

    public String getRegistrationConfirmExitUserNotFoundUrl() {
        return registrationConfirmExitUserNotFoundUrl;
    }

    public String getRegistrationConfirmExitSuccessUrl() {
        return registrationConfirmExitSuccessUrl;
    }

    public String getPasswordRestoreEnterNew() {
        return passwordRestoreEnterNew;
    }

    public String getConfirmChangeEmailExitSuccessUrl() {
        return confirmChangeEmailExitSuccessUrl;
    }

    public String getConfirmChangeEmailExitTokenNotFoundUrl() {
        return confirmChangeEmailExitTokenNotFoundUrl;
    }
}
