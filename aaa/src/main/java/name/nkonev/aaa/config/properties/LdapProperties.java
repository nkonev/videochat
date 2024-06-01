package name.nkonev.aaa.config.properties;

public record LdapProperties(
    LdapAuthProperties auth,
    LdapPasswordEncodingProperties password
) {
}
