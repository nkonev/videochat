package name.nkonev.aaa.security;

import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.dto.UserRole;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.stereotype.Service;

import java.util.Collection;

@Service
public class UserRoleService {

    @Autowired
    private RoleHierarchy roleHierarchy;

    public boolean isAdmin(UserAccountDetailsDTO userAccount) {
        return isAdmin(userAccount.getAuthorities());
    }

    public boolean isAdmin(Collection<? extends GrantedAuthority> authorities) {
        return roleHierarchy.getReachableGrantedAuthorities(authorities).contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
    }
}
