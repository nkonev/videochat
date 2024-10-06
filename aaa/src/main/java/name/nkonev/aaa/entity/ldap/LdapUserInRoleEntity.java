package name.nkonev.aaa.entity.ldap;

import name.nkonev.aaa.dto.ExternalSyncEntity;

public record LdapUserInRoleEntity(
    String id
) implements ExternalSyncEntity {

    @Override
    public String getId() {
        return id;
    }
}
