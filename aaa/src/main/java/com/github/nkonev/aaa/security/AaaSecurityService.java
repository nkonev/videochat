package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.dto.LockDTO;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.dto.UserRole;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.stereotype.Service;

import java.util.Optional;

/**
 * Central entrypoint for access decisions
 */
@Service
public class AaaSecurityService {
    @Autowired
    private RoleHierarchy roleHierarchy;

    @Autowired
    private UserAccountRepository userAccountRepository;

    public boolean hasSessionManagementPermission(UserAccountDetailsDTO userAccount) {
        if (userAccount==null){
            return false;
        }
        return roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }

    public boolean canLock(UserAccountDetailsDTO userAccount, LockDTO lockDTO) {
        if (userAccount==null){
            return false;
        }
        if (lockDTO!=null && userAccount.getId().equals(lockDTO.userId())){
            return false;
        }
        if (roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()))){
            return true;
        } else {
            return false;
        }
    }

    public boolean hasSettingsPermission(UserAccountDetailsDTO userAccount) {
        return Optional
                .ofNullable(userAccount)
                .map(u -> u.getAuthorities()
                        .contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())))
                .orElse(false);
    }

    public boolean canDelete(UserAccountDetailsDTO userAccount, long userIdToDelete) {
        UserAccount deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
        if (deleted.id().equals(userIdToDelete)){
            return false;
        }
        return Optional
                .ofNullable(userAccount)
                .map(u -> u.getAuthorities()
                        .contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())) &&
                        !u.getId().equals(userIdToDelete))
                .orElse(false);
    }

    public boolean canSelfDelete(UserAccountDetailsDTO userAccount) {
        return Optional
                .ofNullable(userAccount).isPresent();
    }

    public boolean canChangeRole(UserAccountDetailsDTO currentUser, long userAccountId) {
        UserAccount userAccount = userAccountRepository.findById(userAccountId).orElseThrow();
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canLock(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canDelete(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canChangeRole(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    private boolean lockAndDelete(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        if (currentUser == null) {
            return  false;
        }
        UserAccount deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
        if (deleted.id().equals(userAccount.id())){
            return false;
        }
        if (userAccount.id().equals(currentUser.getId())){
            return false;
        }
        return roleHierarchy.getReachableGrantedAuthorities(currentUser.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }
}
