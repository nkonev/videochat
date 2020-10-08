package com.github.nkonev.aaa.config;

import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtException;

import static com.github.nkonev.aaa.it.OAuth2EmulatorTests.googleId;
import static com.github.nkonev.aaa.it.OAuth2EmulatorTests.googleLogin;

public class AlwaysTrueJwtDecoder implements JwtDecoder {
    @Override
    public Jwt decode(String
                                  token // actually nonce for test purposes
    ) throws JwtException {
        return Jwt.withTokenValue(token)
                .header("alg", "none")
                .header("typ", "JWT")
                // OAuth2EmulatorTests
                .claim("sub", googleId)
                .claim("aud", "fake-aud")
                .claim("name", googleLogin)
                .claim("admin", true)
                .claim("nonce", token)
                .build();
    }
}
