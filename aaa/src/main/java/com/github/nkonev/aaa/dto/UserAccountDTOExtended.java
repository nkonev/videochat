package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonUnwrapped;

import java.time.LocalDateTime;
import java.util.Set;

public record UserAccountDTOExtended (
    @JsonUnwrapped
    @JsonProperty(access = JsonProperty.Access.READ_ONLY)
    UserAccountDTO userAccountDTO,

    DataDTO additionalData,

    boolean canLock,

    boolean canDelete,

    boolean canChangeRole,
    boolean canConfirm,
    boolean awaitingForConfirmEmailChange
) {

    @JsonCreator
    public UserAccountDTOExtended(
        @JsonProperty("id") Long id,
        @JsonProperty("login") String login,
        @JsonProperty("avatar") String avatar,
        @JsonProperty("avatarBig") String avatarBig,
        @JsonProperty("shortInfo") String shortInfo,
        @JsonProperty("additionalData") DataDTO managementData,
        @JsonProperty("lastLoginDateTime") LocalDateTime lastLoginDateTime,
        @JsonProperty("oauth2Identifiers") OAuth2IdentifiersDTO oauthIdentifiers,
        @JsonProperty("canLock") boolean canLock,
        @JsonProperty("canDelete") boolean canDelete,
        @JsonProperty("canChangeRole") boolean canChangeRole,
        @JsonProperty("canConfirm") boolean canConfirm,
        @JsonProperty("awaitingForConfirmEmailChange") boolean awaitingForConfirmEmailChange
    ) {
        this(
            new UserAccountDTO(id, login, avatar, avatarBig, shortInfo, lastLoginDateTime, oauthIdentifiers),
            managementData,
            canDelete,
            canLock,
            canChangeRole,
            canConfirm,
            awaitingForConfirmEmailChange
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
