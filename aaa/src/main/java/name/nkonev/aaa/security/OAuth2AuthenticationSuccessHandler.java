package name.nkonev.aaa.security;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationSuccessHandler;
import org.springframework.stereotype.Component;
import org.springframework.util.StringUtils;

@Component
public class OAuth2AuthenticationSuccessHandler extends SimpleUrlAuthenticationSuccessHandler {

    public static final String SESSION_ATTR_REDIRECT_URL = "videochat_redirect_url";

    public static final String DEFAULT = "/";

    private static final Logger LOGGER = LoggerFactory.getLogger(OAuth2AuthenticationSuccessHandler.class);

    @Override
    protected String determineTargetUrl(HttpServletRequest request,
                                        HttpServletResponse response) {

        var session = request.getSession(false);
        if (session != null) {
            var a = session.getAttribute(SESSION_ATTR_REDIRECT_URL);
            if (a != null) {
                var url = a.toString();
                if (StringUtils.hasLength(url)) {
                    LOGGER.info("Redirecting user with id {} with addr {} to the restored referer url {}", SecurityUtils.getPrincipal().getId(), request.getHeader("x-real-ip"), url);
                    session.removeAttribute(SESSION_ATTR_REDIRECT_URL);
                    return url;
                }
            }
        }

        return DEFAULT;
    }
}
