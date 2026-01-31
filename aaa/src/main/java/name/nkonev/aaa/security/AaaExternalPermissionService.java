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

        var canCreateBlog = isAdmin;
        var canUnlimitedUpload = isAdmin;
        var canRecordCall = isAdmin;

        if (userAccount.overrideAddPermissions() != null) {
            if (!canCreateBlog) {
                canCreateBlog = has(userAccount.overrideAddPermissions(), ExternalPermission.CAN_CREATE_BLOG);
            }

            if (!canUnlimitedUpload) {
                canUnlimitedUpload = has(userAccount.overrideAddPermissions(), ExternalPermission.CAN_UNLIMITED_UPLOAD);
            }

            if (!canRecordCall) {
                canRecordCall = has(userAccount.overrideAddPermissions(), ExternalPermission.CAN_RECORD_CALL);
            }
        }

        if (userAccount.overrideRemovePermissions() != null) {
            if (canCreateBlog) {
                canCreateBlog = !has(userAccount.overrideRemovePermissions(), ExternalPermission.CAN_CREATE_BLOG);
            }

            if (canUnlimitedUpload) {
                canUnlimitedUpload = !has(userAccount.overrideRemovePermissions(), ExternalPermission.CAN_UNLIMITED_UPLOAD);
            }

            if (canRecordCall) {
                canRecordCall = !has(userAccount.overrideRemovePermissions(), ExternalPermission.CAN_RECORD_CALL);
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
