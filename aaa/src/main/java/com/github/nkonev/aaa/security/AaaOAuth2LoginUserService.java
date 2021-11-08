package com.github.nkonev.aaa.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;

@Component
public class AaaOAuth2LoginUserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    @Autowired
    private FacebookOAuth2UserService facebookOAuth2UserService;

    @Autowired
    private VkontakteOAuth2UserService vkontakteOAuth2UserService;

    @Autowired
    private KeycloakOAuth2UserService keycloakOAuth2UserService;

    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        String clientName = userRequest.getClientRegistration().getRegistrationId();
        switch (clientName) {
            case OAuth2Providers.VKONTAKTE:
                return vkontakteOAuth2UserService.loadUser(userRequest);
            case OAuth2Providers.FACEBOOK:
                return facebookOAuth2UserService.loadUser(userRequest);
            case OAuth2Providers.KEYCLOAK:
                return keycloakOAuth2UserService.loadUser(userRequest);
        }
        throw new RuntimeException("Unknown clientName '" + clientName + "'");
    }
}
