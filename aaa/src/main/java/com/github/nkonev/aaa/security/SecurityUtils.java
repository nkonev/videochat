package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.util.Assert;

public abstract class SecurityUtils {

    /**
     * Set new UserDetails to SecurityContext.
     * When spring mvc finishes request processing, UserDetails will be stored in Session and effectively appears in Redis
     * @param userAccount
     */
    public static void convertAndSetToContext(UserAccount userAccount) {
        Assert.notNull(userAccount, "userAccount cannot be null");
        UserAccountDetailsDTO newUserDetails = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        setToContext(newUserDetails);
    }

    public static void setToContext(UserAccountDetailsDTO newUserDetails) {
        Assert.notNull(SecurityContextHolder.getContext(), "securityContext cannot be null");
        SecurityContextHolder.getContext().setAuthentication(new AaaAuthenticationToken(newUserDetails));
    }
}
