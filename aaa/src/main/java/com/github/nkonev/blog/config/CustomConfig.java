package com.github.nkonev.blog.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

@Configuration
public class CustomConfig {
    @Value("${custom.base-url}")
    private String baseUrl;

    public String getBaseUrl() {
        return baseUrl;
    }
}
