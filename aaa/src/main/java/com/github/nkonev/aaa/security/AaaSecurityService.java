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

import javax.annotation.PostConstruct;
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

    private UserAccount deleted;

    @PostConstruct
    public void postConstruct() {
        deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
    }

    public boolean hasSessionManagementPermission(PrincipalToCheck userAccount) {
        if (userAccount==null){
            return false;
        }
        return userAccount.isAdmin();
    }

    public boolean canLock(PrincipalToCheck userAccount, LockDTO lockDTO) {
        if (userAccount==null){
            return false;
        }
        if (lockDTO!=null && userAccount.getId().equals(lockDTO.userId())){
            return false;
        }
        return userAccount.isAdmin();
    }

    public boolean canDelete(UserAccountDetailsDTO userAccount, long userIdToDelete) {
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

    public boolean canChangeRole(PrincipalToCheck currentUser, long userAccountId) {
        UserAccount userAccount = userAccountRepository.findById(userAccountId).orElseThrow();
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canLock(PrincipalToCheck currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canDelete(PrincipalToCheck currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    public boolean canChangeRole(PrincipalToCheck currentUser, UserAccount userAccount) {
        return lockAndDelete(currentUser, userAccount);
    }

    private boolean lockAndDelete(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        if (currentUser == null) {
            return false;
        }
        if (deleted.id().equals(userAccount.id())){
            return false;
        }
        if (userAccount.id().equals(currentUser.getId())){
            return false;
        }
        return roleHierarchy.getReachableGrantedAuthorities(currentUser.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }
}
