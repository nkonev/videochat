package com.github.nkonev.aaa.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;

@Component
public class AaaOAuth2AuthorizationCodeUserService implements OAuth2UserService<OidcUserRequest, OidcUser> {

    @Autowired
    private GoogleOAuth2UserService googleOAuth2UserService;

    @Override
    public OidcUser loadUser(OidcUserRequest userRequest) throws OAuth2AuthenticationException {
        String clientName = userRequest.getClientRegistration().getRegistrationId();
        switch (clientName) {
            case OAuth2Providers.GOOGLE:
                return googleOAuth2UserService.loadUser(userRequest);
        }
        throw new RuntimeException("Unknown clientName '" + clientName + "'");
    }

}
