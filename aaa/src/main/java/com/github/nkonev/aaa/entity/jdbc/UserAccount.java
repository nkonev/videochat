package com.github.nkonev.aaa.entity.jdbc;

import com.github.nkonev.aaa.dto.UserRole;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Embedded;
import org.springframework.data.relational.core.mapping.Table;
import javax.validation.constraints.NotNull;
import java.time.LocalDateTime;

@Table("users")
public record UserAccount(
    @Id
    Long id,
    String username,
    String password, // hash
    String avatar, // avatar url
    String avatarBig, // avatar url
    boolean expired,
    boolean locked,
    boolean enabled, // synonym to "confirmed"
    String email,

    @NotNull
    CreationType creationType,

    @NotNull
    UserRole role, // synonym to "authority"

    LocalDateTime lastLoginDateTime,

    @Embedded(onEmpty = Embedded.OnEmpty.USE_EMPTY)
    OAuth2Identifiers oauth2Identifiers
) {
    public UserAccount() {
        this(null, null, null, null, null, false, false, false, null, null, null, null, new OAuth2Identifiers(null, null, null));
    }

    public UserAccount(CreationType creationType, String username, String password, String avatar, String avatarBig,
                       boolean expired, boolean locked, boolean enabled,
                       UserRole role, String email, OAuth2Identifiers oauth2Identifiers) {
        this(null, username, password, avatar, avatarBig, expired, locked, enabled, email, creationType, role, null, oauth2Identifiers);
    }

    public UserAccount(CreationType creationType, String username, String password, String avatar, String avatarBig,
                       boolean expired, boolean locked, boolean enabled,
                       UserRole role, String email) {
        this(null, username, password, avatar, avatarBig, expired, locked, enabled, email, creationType, role, null, new OAuth2Identifiers(null, null, null));
    }

    public UserAccount withPassword(String newPassword) {
        return new UserAccount(
                id,
                username,
                newPassword,
                avatar,
                avatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withUsername(String newUsername) {
        return new UserAccount(
                id,
                newUsername,
                password,
                avatar,
                avatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withAvatar(String newAvatar) {
        return new UserAccount(
                id,
                username,
                password,
                newAvatar,
                avatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withAvatarBig(String newAvatarBig) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                newAvatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withEmail(String newEmail) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                avatarBig,
                expired,
                locked,
                enabled,
                newEmail,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withLocked(boolean newLocked) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                avatarBig,
                expired,
                newLocked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withEnabled(boolean newEnabled) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                avatarBig,
                expired,
                locked,
                newEnabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withRole(UserRole newRole) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                avatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                newRole,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withOauthIdentifiers(OAuth2Identifiers newOauthIdentifiers) {
        return new UserAccount(
                id,
                username,
                password,
                avatar,
                avatarBig,
                expired,
                locked,
                enabled,
                email,
                creationType,
                role,
                lastLoginDateTime,
                newOauthIdentifiers
        );
    }

    public record OAuth2Identifiers (
        String facebookId,
        String vkontakteId,
        String googleId
    ) {
        public OAuth2Identifiers withoutFacebookId() {
            return new OAuth2Identifiers(
                    null,
                    vkontakteId,
                    googleId
            );
        }

        public OAuth2Identifiers withoutVkontakteId() {
            return new OAuth2Identifiers(
                    facebookId,
                    null,
                    googleId
            );
        }

        public OAuth2Identifiers withoutGoogleId() {
            return new OAuth2Identifiers(
                    facebookId,
                    vkontakteId,
                    null
            );
        }



        public OAuth2Identifiers withFacebookId(String newFbId) {
            return new OAuth2Identifiers(
                    newFbId,
                    vkontakteId,
                    googleId
            );
        }

        public OAuth2Identifiers withVkontakteId(String newVkId) {
            return new OAuth2Identifiers(
                    facebookId,
                    newVkId,
                    googleId
            );
        }

        public OAuth2Identifiers withGoogleId(String newGid) {
            return new OAuth2Identifiers(
                    facebookId,
                    vkontakteId,
                    newGid
            );
        }
    }

}
