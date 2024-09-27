package name.nkonev.aaa.entity.ldap;

import name.nkonev.aaa.tasks.ExternalSyncEntity;

public record LdapUserInRoleEntity(
    String id
) implements ExternalSyncEntity {

    @Override
    public String getId() {
        return id;
    }
}
