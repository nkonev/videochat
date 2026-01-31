package name.nkonev.aaa.config;

import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.security.SecurityRoleConfig;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import java.util.Collection;
import java.util.Collections;

public class SecurityRoleConfigTest {

    @Test
    public void testAdminCanBeUser() throws Exception {
        SecurityRoleConfig securityRoleConfig = new SecurityRoleConfig();
        Collection<GrantedAuthority> roles = Collections.singletonList(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
        java.util.Collection<? extends GrantedAuthority> reachable = securityRoleConfig.roleHierarchy().getReachableGrantedAuthorities(roles);

        Assertions.assertEquals(2, reachable.size());
        Assertions.assertTrue(reachable.contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())));
        Assertions.assertTrue(reachable.contains(new SimpleGrantedAuthority(UserRole.ROLE_USER.name())));
    }
}
