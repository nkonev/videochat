package name.nkonev.aaa.entity.ldap;

import name.nkonev.aaa.config.properties.LdapAttributes;
import name.nkonev.aaa.tasks.ExternalSyncEntity;

import javax.naming.directory.Attributes;
import java.util.Set;

import static name.nkonev.aaa.utils.ConvertUtils.*;

// all props are nullable
public record LdapEntity(
    String id,
    String username,
    String email,
    Boolean locked,
    Boolean enabled
) implements ExternalSyncEntity {

    public LdapEntity(LdapAttributes attributeNames, Attributes ldapEntry) {
        this(
            extractId(attributeNames, ldapEntry),
            extractUsername(attributeNames, ldapEntry),
            extractEmail(attributeNames, ldapEntry),
            extractLocked(attributeNames, ldapEntry),
            extractEnabled(attributeNames, ldapEntry)
        );
    }

    @Override
    public String getId() {
        return id;
    }
}
