package name.nkonev.aaa.config.properties;

public record LdapProperties(
    LdapAuthProperties auth,
    LdapAttributes attributeNames,
    LdapPasswordEncodingProperties password,
    ConflictResolveStrategy resolveConflictsStrategy,
    LdapGroup group
) {
}
