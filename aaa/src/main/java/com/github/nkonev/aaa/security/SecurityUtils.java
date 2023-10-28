package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;

public abstract class SecurityUtils {
    public static void authenticate(UserAccount userAccount) {
        var auth = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityContext context = SecurityContextHolder.createEmptyContext();
        context.setAuthentication(new AaaAuthenticationToken(auth));
        SecurityContextHolder.setContext(context);
    }
}
