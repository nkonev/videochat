package name.nkonev.aaa.security;

import name.nkonev.aaa.dto.ExternalPermission;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;

@Service
public class AaaExternalPermissionService {

    @Autowired
    private UserRoleService userRoleService;

    public Set<ExternalPermission> evaluatePermissions(UserAccountDetailsDTO userAccount) {
        var isAdmin = userRoleService.isAdmin(userAccount);
        return evaluatePermissions(isAdmin, userAccount.overrideAddPermissions(), userAccount.overrideRemovePermissions());
    }

    public Set<ExternalPermission> evaluatePermissions(boolean isAdmin, ExternalPermission[] overrideAddPermissions, ExternalPermission[] overrideRemovePermissions) {
        var canCreateBlog = isAdmin;
        var canUnlimitedUpload = isAdmin;
        var canRecordCall = isAdmin;

        if (overrideAddPermissions != null) {
            if (!canCreateBlog) {
                canCreateBlog = has(overrideAddPermissions, ExternalPermission.CAN_CREATE_BLOG);
            }

            if (!canUnlimitedUpload) {
                canUnlimitedUpload = has(overrideAddPermissions, ExternalPermission.CAN_UNLIMITED_UPLOAD);
            }

            if (!canRecordCall) {
                canRecordCall = has(overrideAddPermissions, ExternalPermission.CAN_RECORD_CALL);
            }
        }

        if (overrideRemovePermissions != null) {
            if (canCreateBlog) {
                canCreateBlog = !has(overrideRemovePermissions, ExternalPermission.CAN_CREATE_BLOG);
            }

            if (canUnlimitedUpload) {
                canUnlimitedUpload = !has(overrideRemovePermissions, ExternalPermission.CAN_UNLIMITED_UPLOAD);
            }

            if (canRecordCall) {
                canRecordCall = !has(overrideRemovePermissions, ExternalPermission.CAN_RECORD_CALL);
            }
        }

        var res = new HashSet<ExternalPermission>();

        if (canCreateBlog) {
            res.add(ExternalPermission.CAN_CREATE_BLOG);
        }
        if (canUnlimitedUpload) {
            res.add(ExternalPermission.CAN_UNLIMITED_UPLOAD);
        }
        if (canRecordCall) {
            res.add(ExternalPermission.CAN_RECORD_CALL);
        }

        return res;
    }

    private boolean has(ExternalPermission[] overridePermissions, ExternalPermission what) {
        if (overridePermissions == null) {
            return false;
        }

        var set = Arrays.stream(overridePermissions).collect(Collectors.toSet());
        return set.contains(what);
    }
}
