package name.nkonev.aaa.config.properties;

public record LdapPasswordEncodingProperties(
    String encodingType, // password encoding type
    int strength // password strength
) {
}
