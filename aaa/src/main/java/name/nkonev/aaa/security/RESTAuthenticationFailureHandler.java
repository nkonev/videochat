package name.nkonev.aaa.security;

import java.io.IOException;

import org.springframework.security.authentication.AuthenticationServiceException;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationFailureHandler;
import org.springframework.stereotype.Component;

import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

/**
 * Called on wrong credentials
 */
@Component
public class RESTAuthenticationFailureHandler extends SimpleUrlAuthenticationFailureHandler {

    public static final String AAA_AUTH_FAILURE_KEY = "aaa.auth.failure.key";
    public static final String AAA_AUTH_FAILURE_MESSAGE = "aaa.auth.failure.message";

    @Override
    public void onAuthenticationFailure(HttpServletRequest request, HttpServletResponse response,
                                        AuthenticationException exception) throws IOException, ServletException {

        // In order not to expose the original message from non-SpringSecurity exception
        // see SecurityFailedLoginExposingTest
        if (!(exception instanceof AuthenticationServiceException)) {
            // pass further to consume in AaaErrorController
            request.setAttribute(AAA_AUTH_FAILURE_KEY, Boolean.TRUE);
            request.setAttribute(AAA_AUTH_FAILURE_MESSAGE, exception.getMessage());
        }
        super.onAuthenticationFailure(request, response, exception);
    }
}
