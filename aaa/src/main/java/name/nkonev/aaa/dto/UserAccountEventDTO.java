package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import name.nkonev.aaa.Constants;

import java.time.LocalDateTime;

public record UserAccountEventDTO(
        Long id,

        String login,

        String email, // only for user theirself, it shouldn't be shown publicly

        boolean awaitingForConfirmEmailChange,  // only for user theirself, it shouldn't be shown publicly

        String avatar,

        String avatarBig,

        String shortInfo,

        @JsonFormat(shape=JsonFormat.Shape.STRING, pattern= Constants.DATE_FORMAT)
        LocalDateTime lastSeenDateTime,

        OAuth2IdentifiersDTO oauth2Identifiers,
        String loginColor,
        boolean ldap
) {
}
