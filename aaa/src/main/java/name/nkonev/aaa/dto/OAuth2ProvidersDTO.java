package name.nkonev.aaa.dto;

public record OAuth2ProvidersDTO(
        String providerName,
        boolean allowUnbind
) {
}
