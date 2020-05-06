package com.github.nkonev.aaa.config;

import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.security.SecurityPermissionsConfig;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import java.util.Collection;
import java.util.Collections;

public class SecurityPermissionsConfigTest {

    @Test
    public void testAdminCanBeUser() throws Exception {
        SecurityPermissionsConfig securityPermissionsConfig = new SecurityPermissionsConfig();
        Collection<GrantedAuthority> roles = Collections.singletonList(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name()));
        java.util.Collection<? extends GrantedAuthority> reachable = securityPermissionsConfig.roleHierarchy().getReachableGrantedAuthorities(roles);

        Assertions.assertEquals(2, reachable.size());
        Assertions.assertTrue(reachable.contains(new SimpleGrantedAuthority(UserRole.ROLE_ADMIN.name())));
        Assertions.assertTrue(reachable.contains(new SimpleGrantedAuthority(UserRole.ROLE_USER.name())));
    }
}
