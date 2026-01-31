package name.nkonev.aaa.dto;

import java.util.Set;

public record OverriddenPermissionsDTO(
        Set<ExternalPermission> addPermissions,
        Set<ExternalPermission> removePermissions
) {
}
