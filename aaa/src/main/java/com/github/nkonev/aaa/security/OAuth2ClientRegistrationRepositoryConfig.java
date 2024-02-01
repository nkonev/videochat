package com.github.nkonev.aaa.security;

import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientProperties;
import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientPropertiesRegistrationAdapter;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.oauth2.client.registration.ClientRegistration;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.registration.InMemoryClientRegistrationRepository;

import java.util.ArrayList;
import java.util.List;

@EnableConfigurationProperties(OAuth2ClientProperties.class)
@Configuration
public class OAuth2ClientRegistrationRepositoryConfig {
    // Copy-paste from OAuth2ClientRegistrationRepositoryConfiguration
    @Bean
    ClientRegistrationRepository clientRegistrationRepository(OAuth2ClientProperties properties) {
        List<ClientRegistration> registrations = new ArrayList<>(OAuth2ClientPropertiesRegistrationAdapter.getClientRegistrations(properties).values());
        if (registrations.isEmpty()) {
            return registrationId -> null;
        } else {
            return new InMemoryClientRegistrationRepository(registrations);
        }
    }

}
