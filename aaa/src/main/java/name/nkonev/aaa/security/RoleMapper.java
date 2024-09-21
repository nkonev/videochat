package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.RoleMapEntry;
import name.nkonev.aaa.dto.UserRole;
import org.springframework.util.StringUtils;

import java.util.Arrays;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

import static name.nkonev.aaa.utils.RoleUtils.DEFAULT_ROLE;

public abstract class RoleMapper {

    public static Set<UserRole> map(List<RoleMapEntry> roleMappings, Set<String> rawRoles) {
        if (rawRoles == null) {
            return Set.of(DEFAULT_ROLE);
        }
        var roles = rawRoles.stream()
            .filter(StringUtils::hasLength)
            .map(rr -> replaceIfNeed(roleMappings, rr))
            .filter(role -> Arrays.stream(UserRole.values()).map(Enum::name).anyMatch(role::equals))
            .map(UserRole::valueOf)
            .collect(Collectors.toSet());
        if (roles.isEmpty()) {
            return Set.of(DEFAULT_ROLE);
        } else {
            return roles;
        }
    }

    public static UserRole map(List<RoleMapEntry> roleMappings, String their) {
        return UserRole.valueOf(replaceIfNeed(roleMappings, their));
    }

    private static String replaceIfNeed(List<RoleMapEntry> roleMappings, String rawRole) {
        for (RoleMapEntry entry : roleMappings) {
            if (rawRole.equals(entry.their())) {
                return entry.our();
            }
        }
        return rawRole;
    }
}
