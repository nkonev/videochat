package com.github.nkonev.aaa.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserService;
import org.springframework.security.oauth2.client.userinfo.DefaultOAuth2UserService;
import org.springframework.web.client.RestTemplate;

@Configuration
public class OAuth2UserServiceConfig {

    @Autowired
    private RestTemplate restTemplate;

    @Bean
    public DefaultOAuth2UserService defaultOAuth2UserService() {
        DefaultOAuth2UserService defaultOAuth2UserService = new DefaultOAuth2UserService();
        defaultOAuth2UserService.setRestOperations(restTemplate);
        return defaultOAuth2UserService;
    }

    @Bean
    public OidcUserService oidcUserService(DefaultOAuth2UserService defaultOAuth2UserService) {
        OidcUserService oidcUserService = new OidcUserService();
        oidcUserService.setOauth2UserService(defaultOAuth2UserService);
        return oidcUserService;
    }
}
