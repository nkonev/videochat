package name.nkonev.aaa.entity.jdbc;

import name.nkonev.aaa.dto.UserRole;
import jakarta.validation.constraints.NotNull;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;
import java.time.LocalDateTime;

@Table("user_account")
public record UserAccount(

    @Id Long id,
    @NotNull CreationType creationType,
    String login,
    String password, // hash
    String avatar, // avatar url
    String avatarBig, // avatar url

    String shortInfo, // job title, short bio, ...

    boolean expired,
    boolean locked,
    boolean enabled,
    boolean confirmed,
    UserRole[] roles, // synonym to "authority"
    String email,
    LocalDateTime lastSeenDateTime,
    String facebookId,
    String vkontakteId,
    String googleId,
    String keycloakId,
    String ldapId,
    String loginColor,
    LocalDateTime syncLdapDateTime,
    LocalDateTime syncKeycloakDateTime,
    LocalDateTime syncKeycloakRolesDateTime, // actually, it's only for ADMIN role, because already we have USER role
    LocalDateTime syncLdapRolesDateTime // actually, it's only for ADMIN role, because already we have USER role
) {

    public UserAccount withPassword(String newPassword) {
        return new UserAccount(
                id,
                creationType,
                login,
                newPassword,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withLogin(String newLogin) {
        return new UserAccount(
                id,
                creationType,
                newLogin,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withAvatar(String newAvatar) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                newAvatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withAvatarBig(String newAvatarBig) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                newAvatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withEmail(String newEmailToSet) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                newEmailToSet,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withShortInfo(String newShortInfo) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                newShortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withLocked(boolean newLocked) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                newLocked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withEnabled(boolean newEnabled) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                newEnabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withConfirmed(boolean newConfirmed) {
        return new UserAccount(
            id,
            creationType,
            login,
            password,
            avatar,
            avatarBig,
            shortInfo,
            expired,
            locked,
            enabled,
            newConfirmed,
            roles,
            email,
            lastSeenDateTime,
            facebookId,
            vkontakteId,
            googleId,
            keycloakId,
            ldapId,
            loginColor,
            syncLdapDateTime,
            syncKeycloakDateTime,
            syncKeycloakRolesDateTime,
            syncLdapRolesDateTime
        );
    }

    public UserAccount withRoles(UserRole[] newRoles) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                newRoles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withLoginColor(String newLoginColor) {
        return new UserAccount(
            id,
            creationType,
            login,
            password,
            avatar,
            avatarBig,
            shortInfo,
            expired,
            locked,
            enabled,
            confirmed,
            roles,
            email,
            lastSeenDateTime,
            facebookId,
            vkontakteId,
            googleId,
            keycloakId,
            ldapId,
            newLoginColor,
            syncLdapDateTime,
            syncKeycloakDateTime,
            syncKeycloakRolesDateTime,
            syncLdapRolesDateTime
        );
    }

    public UserAccount withSyncLdapTime(LocalDateTime newSyncLdapDateTime) {
        return new UserAccount(
            id,
            creationType,
            login,
            password,
            avatar,
            avatarBig,
            shortInfo,
            expired,
            locked,
            enabled,
            confirmed,
            roles,
            email,
            lastSeenDateTime,
            facebookId,
            vkontakteId,
            googleId,
            keycloakId,
            ldapId,
            loginColor,
            newSyncLdapDateTime,
            syncKeycloakDateTime,
            syncKeycloakRolesDateTime,
            syncLdapRolesDateTime
        );
    }

    public UserAccount withSyncKeycloakTime(LocalDateTime newSyncKeycloakDateTime) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                newSyncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withSyncKeycloakRolesTime(LocalDateTime newSyncKeycloakRolesDateTime) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                newSyncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public UserAccount withSyncLdapRolesTime(LocalDateTime newSyncLdapRolesDateTime) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                facebookId,
                vkontakteId,
                googleId,
                keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                newSyncLdapRolesDateTime
        );
    }

    public UserAccount withOauthIdentifiers(OAuth2Identifiers newOauthIdentifiers) {
        return new UserAccount(
                id,
                creationType,
                login,
                password,
                avatar,
                avatarBig,
                shortInfo,
                expired,
                locked,
                enabled,
                confirmed,
                roles,
                email,
                lastSeenDateTime,
                newOauthIdentifiers.facebookId,
                newOauthIdentifiers.vkontakteId,
                newOauthIdentifiers.googleId,
                newOauthIdentifiers.keycloakId,
                ldapId,
                loginColor,
                syncLdapDateTime,
                syncKeycloakDateTime,
                syncKeycloakRolesDateTime,
                syncLdapRolesDateTime
        );
    }

    public OAuth2Identifiers oauth2Identifiers() {
        return new OAuth2Identifiers(facebookId, vkontakteId, googleId, keycloakId);
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
