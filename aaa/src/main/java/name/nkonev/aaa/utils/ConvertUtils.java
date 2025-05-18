package name.nkonev.aaa.utils;

import name.nkonev.aaa.config.properties.LdapAttributes;
import org.springframework.ldap.support.LdapUtils;
import org.springframework.util.StringUtils;

import javax.naming.NamingEnumeration;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.ldap.Rdn;
import java.util.HashSet;
import java.util.Set;

import static name.nkonev.aaa.converter.UserAccountConverter.*;

public abstract class ConvertUtils {
    public static boolean convertToBoolean(String value) {
        if (value == null) {
            return false;
        }
        value = value.trim();
        if (!StringUtils.hasLength(value)) {
            return false;
        }
        value = value.toLowerCase();
        return value.equals("true") || value.equals("yes") || value.equals("1") || value.contains("enable") || value.contains("ok");
    }

    public static Set<String> convertToStrings(NamingEnumeration rawRoles) {
        try {
            var res = new HashSet<String>();
            while (rawRoles.hasMore()) {
                res.add(NullUtils.getOrNullWrapException(() -> rawRoles.next().toString()));
            }
            return res;
        } catch (NamingException e) {
            throw new RuntimeException(e);
        }
    }

    public static String extractId(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.id())) {
            return null;
        }
        return NullUtils.getOrNullWrapException(() -> ldapEntry.get(attributeNames.id()).get().toString());
    }

    public static String extractUsername(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.username())) {
            return null;
        }
        var ldapUsername = NullUtils.getOrNullWrapException(() -> ldapEntry.get(attributeNames.username()).get().toString());

        return normalizeLogin(ldapUsername);
    }

    public static String extractEmail(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.email())) {
            return null;
        }

        var ldapEmail = NullUtils.getOrNullWrapException(() -> ldapEntry.get(attributeNames.email()).get().toString());

        return normalizeEmail(ldapEmail);
    }

    public static Set<String> extractRoles(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.role())) {
            return null;
        }

        Set<String> rawRoles = new HashSet<>();
        try {
            var t = ldapEntry.get(attributeNames.role());
            if (t == null) {
                return null;
            }
            var groups = t.getAll();
            if (groups != null) {
                rawRoles.addAll(convertToStrings(groups));
            }
            return rawRoles;
        } catch (NamingException e) {
            throw new RuntimeException(e);
        }
    }

    public static Boolean extractLocked(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.locked())) {
            return null;
        }

        var ldapLockedV = NullUtils.getOrNullWrapException(() -> ldapEntry.get(attributeNames.locked()).get().toString());
        if (ldapLockedV == null) {
            return null;
        }
        return convertToBoolean(ldapLockedV);
    }

    public static Boolean extractEnabled(LdapAttributes attributeNames, Attributes ldapEntry) {
        if (!StringUtils.hasLength(attributeNames.enabled())) {
            return null;
        }

        var ldapEnabledV = NullUtils.getOrNullWrapException(() -> ldapEntry.get(attributeNames.enabled()).get().toString());
        if (ldapEnabledV == null) {
            return null;
        }
        return convertToBoolean(ldapEnabledV);
    }

    // extracts user.0 from
    // uid=user.0,ou=People,dc=example,dc=com
    public static String extractExtId(LdapAttributes attributeNames, Object extIdInRole) {
        if (extIdInRole == null) {
            return null;
        }
        var str = extIdInRole.toString();
        var ldapName = LdapUtils.newLdapName(str);
        return ldapName
                .getRdns().stream()
                .filter(rdn -> rdn.getType().equals(attributeNames.id()))
                .findFirst()
                .map(Rdn::getValue)
                .map(Object::toString)
                .orElse(null);
    }
}
