package name.nkonev.aaa.config.properties;

public record LdapAttributes(
    String id,
    String role,
    String email,
    String locked,
    String enabled,
    String username
) {
}
