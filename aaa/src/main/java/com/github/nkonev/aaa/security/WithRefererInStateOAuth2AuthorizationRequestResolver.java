package com.github.nkonev.aaa.security;

import jakarta.servlet.http.HttpServletRequest;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.core.endpoint.OAuth2AuthorizationRequest;
import org.springframework.util.StringUtils;


import static com.github.nkonev.aaa.utils.ServletUtils.getCurrentHttpRequest;

class WithRefererInStateOAuth2AuthorizationRequestResolver implements OAuth2AuthorizationRequestResolver {

    private final DefaultOAuth2AuthorizationRequestResolver delegate;

    public WithRefererInStateOAuth2AuthorizationRequestResolver(DefaultOAuth2AuthorizationRequestResolver delegate) {
        this.delegate = delegate;
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request);
        return patchState(oAuth2AuthorizationRequest);
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request, String clientRegistrationId) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request, clientRegistrationId);
        return patchState(oAuth2AuthorizationRequest);
    }

    private OAuth2AuthorizationRequest patchState(OAuth2AuthorizationRequest auth2AuthorizationRequest) {
        if (auth2AuthorizationRequest == null) {
            return null;
        }
        OAuth2AuthorizationRequest patched = OAuth2AuthorizationRequest.from(auth2AuthorizationRequest).state(auth2AuthorizationRequest.getState()+getSeparatorRefererOrEmpty()).build();
        return patched;
    }

    private String getSeparatorRefererOrEmpty() {
        HttpServletRequest currentHttpRequest = getCurrentHttpRequest();
        if (currentHttpRequest!=null){
            String referer = currentHttpRequest.getHeader("Referer");
            if (StringUtils.hasLength(referer)){
                return OAuth2AuthenticationSuccessHandler.SEPARATOR+referer;
            }
        }
        return "";
    }

}
