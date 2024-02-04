package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import jakarta.servlet.http.HttpSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.util.Assert;

import static org.springframework.security.web.context.HttpSessionSecurityContextRepository.SPRING_SECURITY_CONTEXT_KEY;

public abstract class SecurityUtils {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityUtils.class);

    /**
     * Set new UserDetails to SecurityContext.
     * When spring mvc finishes request processing, UserDetails will be stored in Session and effectively appears in Redis
     * @param userAccount
     */
    public static void convertAndSetToContext(HttpSession httpSession, UserAccount userAccount) {
        Assert.notNull(userAccount, "userAccount cannot be null");
        UserAccountDetailsDTO newUserDetails = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        setToContext(httpSession, newUserDetails);
    }

    public static void setToContext(HttpSession httpSession, UserAccountDetailsDTO newUserDetails) {
        var context = SecurityContextHolder.getContext();
        Assert.notNull(context, "securityContext cannot be null");
        context.setAuthentication(new AaaAuthenticationToken(newUserDetails));
        SecurityContextHolder.setContext(context);
        httpSession.setAttribute(SPRING_SECURITY_CONTEXT_KEY, context);
        LOGGER.info("Successfully set updated SecurityContext to the session");
    }
}
