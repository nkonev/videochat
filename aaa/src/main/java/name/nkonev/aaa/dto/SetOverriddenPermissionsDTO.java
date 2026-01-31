package name.nkonev.aaa.dto;

import java.util.Set;

public record SetOverriddenPermissionsDTO(
    long userId,
    Set<ExternalPermission> addPermissions,
    Set<ExternalPermission> removePermissions
) {
}
