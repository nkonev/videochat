package name.nkonev.aaa.dto;

import java.io.Serializable;

public record OAuth2IdentifiersDTO  (
    String facebookId,
    String vkontakteId,
    String googleId,
    String keycloakId
) implements Serializable {
    public OAuth2IdentifiersDTO() {
        this(null, null, null, null);
    }

    public OAuth2IdentifiersDTO withGoogleId(String newGoogleId) {
        return new OAuth2IdentifiersDTO(
                facebookId,
                vkontakteId,
                newGoogleId,
                keycloakId
        );
    }

    public OAuth2IdentifiersDTO withVkontakteId(String newVkontakteId) {
        return new OAuth2IdentifiersDTO(
                facebookId,
                newVkontakteId,
                googleId,
                keycloakId
        );
    }

    public OAuth2IdentifiersDTO withFacebookId(String newFacebookId) {
        return new OAuth2IdentifiersDTO(
                newFacebookId,
                vkontakteId,
                googleId,
                keycloakId
        );
    }

    public OAuth2IdentifiersDTO withKeycloakId(String newKeycloakId) {
        return new OAuth2IdentifiersDTO(
                facebookId,
                vkontakteId,
                googleId,
                newKeycloakId
        );
    }
}
