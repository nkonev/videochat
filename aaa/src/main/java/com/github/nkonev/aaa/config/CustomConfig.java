package com.github.nkonev.aaa.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

import java.time.Duration;

@Configuration
public class CustomConfig {
    @Value("${custom.base-url}")
    private String baseUrl;

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
}
