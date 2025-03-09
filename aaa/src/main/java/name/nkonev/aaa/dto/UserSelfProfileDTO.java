package name.nkonev.aaa.dto;

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
        boolean awaitingForConfirmEmailChange,

        Collection<UserRole> roles,

        // session expires at
        Long expiresAt,
        boolean canShowAdminsCorner
) {
    public UserSelfProfileDTO(
            Long id,
            String login,
            String avatar,
            String avatarBig,
            String shortInfo,
            String email,
            boolean awaitingForConfirmEmailChange,
            LocalDateTime lastSeenDateTime,
            OAuth2IdentifiersDTO oauth2Identifiers,
            Collection<UserRole> roles,
            Long expiresAt,
            String loginColor,
            boolean ldap,
            boolean canShowAdminsCorner
    ) {
        this(new UserAccountDTO(
                id,
                login,
                avatar,
                avatarBig,
                shortInfo,
                lastSeenDateTime,
                oauth2Identifiers,
                loginColor,
                ldap
        ), email, awaitingForConfirmEmailChange, roles, expiresAt, canShowAdminsCorner);
    }

    public String login() {
        return userAccountDTO.login();
    }
}
