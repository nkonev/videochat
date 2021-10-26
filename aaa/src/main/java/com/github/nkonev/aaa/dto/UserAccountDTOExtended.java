package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

import java.time.LocalDateTime;

public record UserAccountDTOExtended (
    @JsonUnwrapped
    UserAccountDTO userAccountDTO,

    DataDTO managementData,

    boolean canLock,

    boolean canDelete,

    boolean canChangeRole
) {

    public UserAccountDTOExtended(Long id, String login, String avatar, String avatarBig, DataDTO managementData, LocalDateTime lastLoginDateTime, OAuth2IdentifiersDTO oauthIdentifiers, boolean canLock, boolean canDelete, boolean canChangeRole) {
        this(
            new UserAccountDTO(id, login, avatar, avatarBig, lastLoginDateTime, oauthIdentifiers),
            managementData,
            canDelete,
            canLock,
            canChangeRole
        );
    }

    public record DataDTO (
        boolean enabled,
        boolean expired,
        boolean locked,
        UserRole role
    ) { }

}
