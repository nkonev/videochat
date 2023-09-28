package com.github.nkonev.aaa.services;

import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientProperties;
import org.springframework.stereotype.Service;

import java.util.Set;
import java.util.stream.Collectors;

import static java.util.stream.Stream.ofNullable;

@Service
public class OAuth2ProvidersService {
    @Autowired
    private ObjectProvider<OAuth2ClientProperties> oAuth2ClientProperties;

    public Set<String> availableOauth2Providers() {
        return ofNullable(oAuth2ClientProperties.getIfAvailable())
            .map(OAuth2ClientProperties::getRegistration)
            .flatMap(stringRegistrationMap -> stringRegistrationMap.keySet().stream())
            .collect(Collectors.toSet());
    }

}
