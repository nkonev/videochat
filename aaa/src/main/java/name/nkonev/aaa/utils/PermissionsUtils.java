package name.nkonev.aaa.utils;

import name.nkonev.aaa.dto.ExternalPermission;

public abstract class PermissionsUtils {
    public static boolean areOverriddenPermissions(ExternalPermission[] overrideAddPermissions, ExternalPermission[] overrideRemovePermissions) {
        return has(overrideAddPermissions) || has(overrideRemovePermissions);
    }

    private static boolean has(ExternalPermission[] permissions) {
        return permissions != null && permissions.length > 0;
    }
}
