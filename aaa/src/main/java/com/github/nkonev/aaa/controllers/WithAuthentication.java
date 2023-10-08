package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.security.AaaAuthenticationToken;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;

public abstract class WithAuthentication {
    protected void authenticate(UserAccount userAccount) {
        var auth = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityContext context = SecurityContextHolder.createEmptyContext();
        context.setAuthentication(new AaaAuthenticationToken(auth));
        SecurityContextHolder.setContext(context);
    }

}
