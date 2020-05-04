package com.github.nkonev.blog.dto;

import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.oauth2.core.user.OAuth2User;

import java.net.URL;
import java.time.LocalDateTime;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;

/**
 * Internal class for Spring Security, it shouldn't be passed to browser via Rest API
 */
public class UserAccountDetailsDTO extends UserAccountDTO implements UserDetails, OAuth2User {
    private static final long serialVersionUID = -3271989114498135073L;

    private Map<String, Object> oauth2Attributes = new HashMap<>();

    private String password; // password hash
    private boolean expired;
    private boolean locked;
    private boolean enabled; // synonym to "confirmed"

    private Collection<GrantedAuthority> roles = new HashSet<>();
    private String email;

    public UserAccountDetailsDTO() { }

    public UserAccountDetailsDTO(
            Long id,
            String login,
            String avatar,
            String password,
            boolean expired,
            boolean locked,
            boolean enabled,
            Collection<GrantedAuthority> roles,
            String email,
            LocalDateTime lastLoginDateTime,
            OauthIdentifiersDTO oauthIdentifiers
    ) {
        super(id, login, avatar, lastLoginDateTime, oauthIdentifiers);
        this.password = password;
        this.expired = expired;
        this.locked = locked;
        this.enabled = enabled;
        this.roles = roles;
        this.email = email;
    }

    @Override
    public String getPassword() {
        return password;
    }

    @Override
    public String getUsername() {
        return super.getLogin();
    }

    @Override
    public boolean isAccountNonExpired() {
        return !expired;
    }

    @Override
    public boolean isAccountNonLocked() {
        return !locked;
    }

    @Override
    public boolean isCredentialsNonExpired() {
        return true;
    }

    @Override
    public boolean isEnabled() {
        return enabled;
    }

    @Override
    public Map<String, Object> getAttributes() {
        return oauth2Attributes;
    }

    @Override
    public Collection<? extends GrantedAuthority> getAuthorities() {
        return roles;
    }

    @Override
    public String getName() {
        return login;
    }

    public void setPassword(String password) {
        this.password = password;
    }

    public boolean isExpired() {
        return expired;
    }

    public void setExpired(boolean expired) {
        this.expired = expired;
    }

    public boolean isLocked() {
        return locked;
    }

    public void setLocked(boolean locked) {
        this.locked = locked;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public Collection<GrantedAuthority> getRoles() {
        return roles;
    }

    public void setRoles(Collection<GrantedAuthority> roles) {
        this.roles = roles;
    }

    public String getEmail() {
        return email;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    public Map<String, Object> getOauth2Attributes() {
        return oauth2Attributes;
    }

    public void setOauth2Attributes(Map<String, Object> oauth2Attributes) {
        this.oauth2Attributes = oauth2Attributes;
    }

}
