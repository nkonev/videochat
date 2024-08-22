package name.nkonev.aaa.config.properties;

import java.util.List;

public record RoleMappings(
    List<RoleMapEntry> ldap,
    List<RoleMapEntry> keycloak
) {
}
