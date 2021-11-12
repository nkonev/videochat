package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.security.authentication.AbstractAuthenticationToken;

public class AaaAuthenticationToken extends AbstractAuthenticationToken {

    private final UserAccountDetailsDTO userAccountDetailsDTO;

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
