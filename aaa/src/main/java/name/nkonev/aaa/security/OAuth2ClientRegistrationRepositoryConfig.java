package name.nkonev.aaa.security;

import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.registration.InMemoryClientRegistrationRepository;
import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientPropertiesMapper;

/**
 * To have a possibility to run aaa without any OAuth 2.0 configured
 */
@EnableConfigurationProperties(OAuth2ClientProperties.class)
@Configuration
public class OAuth2ClientRegistrationRepositoryConfig {
    // Copy-paste from OAuth2ClientRegistrationRepositoryConfiguration
    @Bean
    ClientRegistrationRepository clientRegistrationRepository(OAuth2ClientProperties properties) {
        var factory = new OAuth2ClientPropertiesMapper(properties);
        var registrations = factory.asClientRegistrations();
        if (registrations.isEmpty()) {
            return registrationId -> null;
        } else {
            return new InMemoryClientRegistrationRepository(registrations);
        }
    }

}
