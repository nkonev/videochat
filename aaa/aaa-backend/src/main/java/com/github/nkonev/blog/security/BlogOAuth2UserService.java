package com.github.nkonev.blog.security;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;

@Component
public class BlogOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    @Autowired
    private FacebookOAuth2UserService facebookOAuth2UserService;

    @Autowired
    private VkontakteOAuth2UserService vkontakteOAuth2UserService;

    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        String clientName = userRequest.getClientRegistration().getClientName();
        switch (clientName) {
            case "vkontakte":
                return vkontakteOAuth2UserService.loadUser(userRequest);
            case "facebook":
                return facebookOAuth2UserService.loadUser(userRequest);
        }
        throw new RuntimeException("Unknown clientName '" + clientName + "'");
    }
}
