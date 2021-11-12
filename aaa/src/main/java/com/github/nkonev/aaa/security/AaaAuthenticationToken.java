package com.github.nkonev.aaa.security;

import com.fasterxml.jackson.annotation.JsonAutoDetect;
import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonTypeInfo;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.security.authentication.AbstractAuthenticationToken;

@JsonTypeInfo(use = JsonTypeInfo.Id.CLASS, include = JsonTypeInfo.As.PROPERTY, property = "@class")
@JsonAutoDetect(fieldVisibility = JsonAutoDetect.Visibility.ANY, getterVisibility = JsonAutoDetect.Visibility.NONE, setterVisibility = JsonAutoDetect.Visibility.NONE, isGetterVisibility = JsonAutoDetect.Visibility.NONE)
public class AaaAuthenticationToken extends AbstractAuthenticationToken {

    private final UserAccountDetailsDTO userAccountDetailsDTO;

    @JsonCreator
    public AaaAuthenticationToken(UserAccountDetailsDTO userAccountDetailsDTO) {
        super(userAccountDetailsDTO.getAuthorities());
        this.userAccountDetailsDTO = userAccountDetailsDTO;
    }

    @Override
    public Object getCredentials() {
        return userAccountDetailsDTO.getPassword();
    }

    @Override
    public Object getPrincipal() {
        return userAccountDetailsDTO;
    }
}
