package name.nkonev.aaa.dto;

import java.util.List;

public record ConfigDTO(
        List<OAuth2ProvidersDTO> providers,
        long frontendSessionPingInterval,
        int minPasswordLength,
        int maxPasswordLength
) {
}
