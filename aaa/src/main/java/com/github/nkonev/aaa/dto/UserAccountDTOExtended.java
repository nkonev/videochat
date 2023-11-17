package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

import java.time.LocalDateTime;
import java.util.Set;

public record UserAccountDTOExtended (
    @JsonUnwrapped
    UserAccountDTO userAccountDTO,

    DataDTO additionalData,

    boolean canLock,

    boolean canDelete,

    boolean canChangeRole
) {

    public UserAccountDTOExtended(Long id, String login, String avatar, String avatarBig, String shortInfo, DataDTO managementData, LocalDateTime lastLoginDateTime, OAuth2IdentifiersDTO oauthIdentifiers, boolean canLock, boolean canDelete, boolean canChangeRole) {
        this(
            new UserAccountDTO(id, login, avatar, avatarBig, shortInfo, lastLoginDateTime, oauthIdentifiers),
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
        boolean confirmed,
        Set<UserRole> roles
    ) { }

}
