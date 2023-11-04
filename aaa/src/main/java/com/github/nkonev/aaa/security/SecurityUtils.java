package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;

public abstract class SecurityUtils {

    public static void authenticate(UserAccountDetailsDTO auth) {
        SecurityContext context = SecurityContextHolder.createEmptyContext();
        context.setAuthentication(new AaaAuthenticationToken(auth));
        SecurityContextHolder.setContext(context);
    }

}
