package com.github.nkonev.aaa.security.converter;

import org.springframework.core.convert.converter.Converter;
import org.springframework.security.oauth2.core.OAuth2AccessToken;
import org.springframework.security.oauth2.core.endpoint.DefaultMapOAuth2AccessTokenResponseConverter;
import org.springframework.security.oauth2.core.endpoint.OAuth2AccessTokenResponse;
import org.springframework.security.oauth2.core.endpoint.OAuth2ParameterNames;
import java.util.*;

public class BearerOAuth2AccessTokenResponseConverter implements Converter<Map<String, Object>, OAuth2AccessTokenResponse> {

    private final DefaultMapOAuth2AccessTokenResponseConverter delegate = new DefaultMapOAuth2AccessTokenResponseConverter();

    @Override
    public OAuth2AccessTokenResponse convert(Map<String, Object> tokenResponseParameters) {
        // vkontakte doesn't pass token type in response so I add it manually
        tokenResponseParameters.putIfAbsent(OAuth2ParameterNames.TOKEN_TYPE, OAuth2AccessToken.TokenType.BEARER.getValue());
        return delegate.convert(tokenResponseParameters);
    }
}
