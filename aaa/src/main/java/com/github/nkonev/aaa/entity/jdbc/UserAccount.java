package com.github.nkonev.aaa.entity.jdbc;

import com.github.nkonev.aaa.dto.UserRole;
import org.springframework.data.annotation.Id;
import org.springframework.data.annotation.PersistenceConstructor;
import org.springframework.data.relational.core.mapping.Embedded;
import org.springframework.data.relational.core.mapping.Table;
import javax.validation.constraints.NotNull;
import java.time.LocalDateTime;

@Table("user_account")
public record UserAccount(

    @Id Long id,
    @NotNull CreationType creationType,
    String username,
    String password, // hash
    String avatar, // avatar url
    String avatarBig, // avatar url

    String shortInfo, // job title, short bio, ...

    boolean expired,
    boolean locked,
    boolean enabled,
    boolean confirmed,
    @NotNull UserRole role, // synonym to "authority"
    String email,
    LocalDateTime lastLoginDateTime,
    @Embedded(onEmpty = Embedded.OnEmpty.USE_EMPTY) OAuth2Identifiers oauth2Identifiers
) {

    public UserAccount withPassword(String newPassword) {
        return new UserAccount(
                id,
                creationType,
                username,
                newPassword,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withUsername(String newUsername) {
        return new UserAccount(
                id,
                creationType,
                newUsername,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withAvatar(String newAvatar) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                newAvatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withAvatarBig(String newAvatarBig) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                newAvatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withEmail(String newEmail) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                newEmail,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withShortInfo(String newShortInfo) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                newShortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withLocked(boolean newLocked) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                newLocked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withEnabled(boolean newEnabled) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                newEnabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withConfirmed(boolean newConfirmed) {
        return new UserAccount(
            id,
            creationType,
            username,
            password,
            avatar,
            avatarBig,
            shortInfo,
            expired,
            locked,
            enabled,
            newConfirmed,
            role,
            email,
            lastLoginDateTime,
            oauth2Identifiers
        );
    }

    public UserAccount withRole(UserRole newRole) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                newRole,
                email,
                lastLoginDateTime,
                oauth2Identifiers
        );
    }

    public UserAccount withOauthIdentifiers(OAuth2Identifiers newOauthIdentifiers) {
        return new UserAccount(
                id,
                creationType,
                username,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                role,
                email,
                lastLoginDateTime,
                newOauthIdentifiers
        );
    }

    @Override
    public OAuth2Identifiers oauth2Identifiers() {
        return oauth2Identifiers != null ? oauth2Identifiers : new OAuth2Identifiers(null, null, null, null);
    }

    public record OAuth2Identifiers (
        String facebookId,
        String vkontakteId,
        String googleId,
        String keycloakId
    ) {
        public OAuth2Identifiers withFacebookId(String newFbId) {
            return new OAuth2Identifiers(
                    newFbId,
                    vkontakteId,
                    googleId,
                    keycloakId
            );
        }

        public OAuth2Identifiers withVkontakteId(String newVkId) {
            return new OAuth2Identifiers(
                    facebookId,
                    newVkId,
                    googleId,
                    keycloakId
            );
        }

        public OAuth2Identifiers withGoogleId(String newGid) {
            return new OAuth2Identifiers(
                    facebookId,
                    vkontakteId,
                    newGid,
                    keycloakId
            );
        }

        public OAuth2Identifiers withKeycloakId(String newKid) {
            return new OAuth2Identifiers(
                    facebookId,
                    vkontakteId,
                    googleId,
                    newKid
            );
        }

    }

}
