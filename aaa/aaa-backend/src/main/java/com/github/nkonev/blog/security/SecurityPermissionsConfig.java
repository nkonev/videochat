package com.github.nkonev.blog.security;

import com.github.nkonev.blog.dto.UserRole;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.access.expression.method.DefaultMethodSecurityExpressionHandler;
import org.springframework.security.access.expression.method.MethodSecurityExpressionHandler;
import org.springframework.security.access.hierarchicalroles.RoleHierarchy;
import org.springframework.security.access.hierarchicalroles.RoleHierarchyImpl;
import org.springframework.security.config.annotation.method.configuration.EnableGlobalMethodSecurity;

// https://docs.spring.io/spring-security/site/docs/current/reference/htmlsingle/#domain-acls
// https://stackoverflow.com/questions/26292431/how-to-configure-spring-acl-without-xml-file
// gotcha https://stackoverflow.com/questions/38609874/acl-security-in-spring-boot
// http://book2s.com/java/src/package/com/foreach/across/modules/spring/security/acl/config/aclsecurityconfiguration.html

/**
 * tables in auth schema
 * ACL_SID              GrantedAuthority(ROLE_ADMIN, ...) or Principal
 * ACL_CLASS            Post.class or Comment.class
 * ACL_OBJECT_IDENTITY  acl_class_id, parent_acl_sid_id(owner) Post or Comment instance
 * ACL_ENTRY            acl_object_identity_id, acl_sid_id(recipient), auditing, permissions_bitmask_integer
 */
@Configuration
@EnableGlobalMethodSecurity(prePostEnabled = true, securedEnabled = true)
public class SecurityPermissionsConfig {

    // @Override
    @Bean
    public MethodSecurityExpressionHandler methodSecurityExpressionHandler(){
        DefaultMethodSecurityExpressionHandler expressionHandler = new DefaultMethodSecurityExpressionHandler();
        // expressionHandler.setPermissionEvaluator(blogPermissionEvaluator);
        expressionHandler.setRoleHierarchy(roleHierarchy());
        return expressionHandler;
    }

    @Bean
    public RoleHierarchy roleHierarchy(){
        RoleHierarchyImpl roleHierarchy = new RoleHierarchyImpl();
        roleHierarchy.setHierarchy(
                UserRole.ROLE_ADMIN.name() + " > " + UserRole.ROLE_MODERATOR.name() + "\n"+
                UserRole.ROLE_MODERATOR.name() + " > " + UserRole.ROLE_USER.name() + "\n"
        );
        return roleHierarchy;
    }
}
