package name.nkonev.aaa.security;

import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
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
    public static void convertAndSetToContext(UserAccountConverter userAccountConverter, HttpSession httpSession, UserAccount userAccount) {
        Assert.notNull(userAccount, "userAccount cannot be null");
        UserAccountDetailsDTO newUserDetails = userAccountConverter.convertToUserAccountDetailsDTO(userAccount);
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

    // can throw an exception in case no context
    public static UserAccountDetailsDTO getPrincipal() {
        return (UserAccountDetailsDTO) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
    }
}
