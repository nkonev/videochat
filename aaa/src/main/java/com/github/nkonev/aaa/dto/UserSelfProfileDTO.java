package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

import java.time.LocalDateTime;
import java.util.Collection;

/**
 * Class which displays in user's profile page. It will be POSTed as EditUserDTO
 */
public record UserSelfProfileDTO(
        @JsonUnwrapped
        UserAccountDTO userAccountDTO,
        String email,

        Collection<UserRole> roles,

        // session expires at
        Long expiresAt
) {
    public UserSelfProfileDTO(
            Long id,
            String username,
            String avatar,
            String avatarBig,
            String email,
            LocalDateTime lastLoginDateTime,
            OAuth2IdentifiersDTO oauth2Identifiers,
            Collection<UserRole> roles,
            Long expiresAt
    ) {
        this(new UserAccountDTO(
                id,
                username,
                avatar,
                avatarBig,
                lastLoginDateTime,
                oauth2Identifiers
        ), email, roles, expiresAt);
    }

    public String login() {
        return userAccountDTO.login();
    }
}