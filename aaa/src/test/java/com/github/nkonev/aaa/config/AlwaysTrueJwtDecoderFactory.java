package com.github.nkonev.aaa.config;

import org.springframework.security.oauth2.client.registration.ClientRegistration;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtDecoderFactory;
import org.springframework.stereotype.Component;

@Component
public class AlwaysTrueJwtDecoderFactory implements JwtDecoderFactory<ClientRegistration> {
    @Override
    public JwtDecoder createDecoder(ClientRegistration context) {
        return new AlwaysTrueJwtDecoder();
    }
}
