package com.github.nkonev.aaa.security;

import com.fasterxml.jackson.annotation.JsonAutoDetect;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.GrantedAuthority;

import java.util.Collection;

@JsonAutoDetect(fieldVisibility = JsonAutoDetect.Visibility.ANY, getterVisibility = JsonAutoDetect.Visibility.NONE, setterVisibility = JsonAutoDetect.Visibility.NONE, isGetterVisibility = JsonAutoDetect.Visibility.NONE)
public class AaaAuthenticationToken implements Authentication {

    private UserAccountDetailsDTO userAccountDetailsDTO;

    public AaaAuthenticationToken() {}

    public AaaAuthenticationToken(UserAccountDetailsDTO userAccountDetailsDTO) {
        this.userAccountDetailsDTO = userAccountDetailsDTO;
    }

    @Override
    public Collection<? extends GrantedAuthority> getAuthorities() {
        return userAccountDetailsDTO.getAuthorities();
    }

    @Override
    public Object getCredentials() {
        return userAccountDetailsDTO.getPassword();
    }

    @Override
    public Object getDetails() {
        return userAccountDetailsDTO;
    }

    @Override
    public Object getPrincipal() {
        return getDetails();
    }

    @Override
    public boolean isAuthenticated() {
        return true;
    }

    @Override
    public void setAuthenticated(boolean isAuthenticated) throws IllegalArgumentException {
        if (!isAuthenticated) {
            throw new IllegalArgumentException("Not possible");
        }
    }

    @Override
    public String getName() {
        return userAccountDetailsDTO.getName();
    }
}
