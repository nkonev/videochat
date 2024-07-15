package name.nkonev.aaa.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.services.EventService;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.core.Authentication;
import org.springframework.security.web.authentication.logout.LogoutSuccessHandler;
import org.springframework.security.web.csrf.CsrfToken;
import org.springframework.security.web.csrf.CsrfTokenRepository;
import org.springframework.stereotype.Component;
import org.springframework.util.Assert;
import java.io.IOException;
import java.util.Collections;
import java.util.List;

/**
 * Created by nik on 09.07.17.
 *
 * This listener works for both database and OAuth2 logouts
 */
@Component
public class RESTAuthenticationLogoutSuccessHandler implements LogoutSuccessHandler {

    private final ObjectProvider<CsrfTokenRepository> csrfTokenRepositoryProvider;

    private final ObjectMapper objectMapper;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private EventService eventService;

    private static final Logger LOGGER = LoggerFactory.getLogger(RESTAuthenticationLogoutSuccessHandler.class);

    public RESTAuthenticationLogoutSuccessHandler(ObjectProvider<CsrfTokenRepository> csrfTokenRepositoryProvider, ObjectMapper objectMapper) {
        Assert.notNull(csrfTokenRepositoryProvider, "csrfTokenRepository cannot be null");
        Assert.notNull(objectMapper, "objectMapper cannot be null");
        this.csrfTokenRepositoryProvider = csrfTokenRepositoryProvider;
        this.objectMapper = objectMapper;
    }


    @Override
    public void onLogoutSuccess(HttpServletRequest request, HttpServletResponse response, Authentication authentication) throws IOException, ServletException {
        // do nothing -- it's enough to return 200 for SPA

        // set new csrf token for repeating logins without page reload
        var csrfTokenRepository = csrfTokenRepositoryProvider.getObject();
        CsrfToken csrfToken = csrfTokenRepository.generateToken(request);
        csrfTokenRepository.saveToken(csrfToken, request, response);

        UserAccountDetailsDTO userDetails = (UserAccountDetailsDTO)authentication.getPrincipal();
        LOGGER.info("User '{}' logged out", userDetails.getLogin());

        var usersOnline = aaaUserDetailsService.getUsersOnline(List.of(userDetails.getId()));
        eventService.notifyOnlineChanged(usersOnline);

        response.setContentType(MediaType.APPLICATION_JSON_UTF8_VALUE);
        objectMapper.writeValue(response.getWriter(), Collections.singletonMap("message", "you successfully logged out"));
    }
}
