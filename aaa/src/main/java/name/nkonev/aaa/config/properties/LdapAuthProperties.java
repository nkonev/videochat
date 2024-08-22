package name.nkonev.aaa.config.properties;

public record LdapAuthProperties(
    String base,
    boolean enabled,
    String filter
) {
}
