package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.dto.UserRole;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.core.authority.SimpleGrantedAuthority;

import static com.github.nkonev.aaa.Constants.NonExistentUser;

public sealed interface PrincipalToCheck permits KnownAdmin, UserToCheck {
    boolean isAdmin();

    Long getId();

    static PrincipalToCheck knownAdmin() {
        return new KnownAdmin();
    }

    static PrincipalToCheck ofUserAccount(UserAccountDetailsDTO userAccount, RoleHierarchy roleHierarchy) {
        return new UserToCheck(userAccount, roleHierarchy);
    }
}

final class KnownAdmin implements PrincipalToCheck {

    @Override
    public boolean isAdmin() {
        return true;
    }

    @Override
    public Long getId() {
        return NonExistentUser;
    }
}

final class UserToCheck implements PrincipalToCheck {

    private final UserAccountDetailsDTO userAccount;

    private final RoleHierarchy roleHierarchy;

    UserToCheck(UserAccountDetailsDTO userAccount, RoleHierarchy roleHierarchy) {
        this.userAccount = userAccount;
        this.roleHierarchy = roleHierarchy;
    }

    @Override
    public boolean isAdmin() {
        return roleHierarchy.getReachableGrantedAuthorities(userAccount.getAuthorities()).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }

    @Override
    public Long getId() {
        return userAccount.getId();
    }
}
