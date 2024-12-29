package name.nkonev.aaa.dto;

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

    // can I as an user / admin lock him ?
    boolean canLock,
    boolean canEnable,

    boolean canDelete,

    boolean canChangeRole,
    boolean canConfirm,
    boolean awaitingForConfirmEmailChange,
    boolean canRemoveSessions,
    boolean canSetPassword
) {

    @JsonCreator
    public UserAccountDTOExtended(
        @JsonProperty("id") Long id,
        @JsonProperty("login") String login,
        @JsonProperty("avatar") String avatar,
        @JsonProperty("avatarBig") String avatarBig,
        @JsonProperty("shortInfo") String shortInfo,
        @JsonProperty("additionalData") DataDTO managementData,
        @JsonProperty("lastSeenDateTime") LocalDateTime lastSeenDateTime,
        @JsonProperty("oauth2Identifiers") OAuth2IdentifiersDTO oauthIdentifiers,
        @JsonProperty("canLock") boolean canLock,
        @JsonProperty("canEnable") boolean canEnable,
        @JsonProperty("canDelete") boolean canDelete,
        @JsonProperty("canChangeRole") boolean canChangeRole,
        @JsonProperty("canConfirm") boolean canConfirm,
        @JsonProperty("awaitingForConfirmEmailChange") boolean awaitingForConfirmEmailChange,
        @JsonProperty("loginColor") String loginColor,
        @JsonProperty("canRemoveSessions") boolean canRemoveSessions,
        @JsonProperty("ldap") boolean ldap,
        @JsonProperty("canSetPassword") boolean canSetPassword
    ) {
        this(
            new UserAccountDTO(id, login, avatar, avatarBig, shortInfo, lastSeenDateTime, oauthIdentifiers, loginColor, ldap),
            managementData,
            canDelete,
            canLock,
            canEnable,
            canChangeRole,
            canConfirm,
            awaitingForConfirmEmailChange,
            canRemoveSessions,
            canSetPassword
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
