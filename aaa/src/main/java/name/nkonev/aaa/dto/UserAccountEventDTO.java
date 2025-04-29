package name.nkonev.aaa.dto;

import java.time.LocalDateTime;

public record UserAccountEventDTO(
        Long id,

        String login,

        String email, // only for user theirself, it shouldn't be shown publicly

        boolean awaitingForConfirmEmailChange,  // only for user theirself, it shouldn't be shown publicly

        String avatar,

        String avatarBig,

        String shortInfo,

        LocalDateTime lastSeenDateTime,

        OAuth2IdentifiersDTO oauth2Identifiers,
        String loginColor,
        boolean ldap,
        AdditionalDataDTO additionalData
) {
}
