package name.nkonev.aaa.services;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.OAuth2ProvidersDTO;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.security.oauth2.client.autoconfigure.OAuth2ClientProperties;
import org.springframework.stereotype.Service;

import java.util.List;

import static java.util.stream.Stream.ofNullable;
import static name.nkonev.aaa.security.OAuth2Providers.KEYCLOAK;

@Service
public class OAuth2ProvidersService {
    @Autowired
    private ObjectProvider<OAuth2ClientProperties> oAuth2ClientProperties;

    @Autowired
    private AaaProperties aaaProperties;

    public List<OAuth2ProvidersDTO> availableOauth2Providers() {
        return ofNullable(oAuth2ClientProperties.getIfAvailable())
            .map(OAuth2ClientProperties::getRegistration)
            .flatMap(stringRegistrationMap -> stringRegistrationMap.keySet().stream())
            .map(n -> new OAuth2ProvidersDTO(n, isAllowDetach(n)))
            .toList();
    }

    private boolean isAllowDetach(String providerName) {
        if (KEYCLOAK.equals(providerName)) {
            return aaaProperties.keycloak().allowUnbind();
        }
        return true;
    }

}
