package name.nkonev.aaa.security;

import name.nkonev.aaa.exception.OAuth2IdConflictException;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationFailureHandler;
import org.springframework.stereotype.Component;

import java.io.IOException;

@Component
public class OAuth2ExceptionHandler extends SimpleUrlAuthenticationFailureHandler {

    private static final Logger LOGGER = LoggerFactory.getLogger(OAuth2ExceptionHandler.class);

    @Override
    public void onAuthenticationFailure(HttpServletRequest request,
                                        HttpServletResponse response, AuthenticationException exception)
            throws IOException, ServletException {

        if (exception instanceof OAuth2IdConflictException e){
            var m = e.getMessage();
            LOGGER.info("Handling OAuth2IdConflictException, message: {}", m);
            response.getOutputStream().println(m);
            response.setStatus(403);
        } else {
            super.onAuthenticationFailure(request, response, exception);
        }
    }

}
