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
        Long expiresAt
) {
    public UserSelfProfileDTO(
            Long id,
            String username,
            String avatar,
            String avatarBig,
            String shortInfo,
            String email,
            boolean awaitingForConfirmEmailChange,
            LocalDateTime lastLoginDateTime,
            OAuth2IdentifiersDTO oauth2Identifiers,
            Collection<UserRole> roles,
            Long expiresAt,
            String loginColor,
            boolean ldap
    ) {
        this(new UserAccountDTO(
                id,
                username,
                avatar,
                avatarBig,
                shortInfo,
                lastLoginDateTime,
                oauth2Identifiers,
                loginColor,
                ldap
        ), email, awaitingForConfirmEmailChange, roles, expiresAt);
    }

    public String login() {
        return userAccountDTO.login();
    }
}
