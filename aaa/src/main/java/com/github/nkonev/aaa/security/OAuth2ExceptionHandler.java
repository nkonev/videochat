package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.exception.OAuth2IdConflictException;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationFailureHandler;
import org.springframework.stereotype.Component;

import java.io.IOException;

@Component
public class OAuth2ExceptionHandler extends SimpleUrlAuthenticationFailureHandler {
    @Override
    public void onAuthenticationFailure(HttpServletRequest request,
                                        HttpServletResponse response, AuthenticationException exception)
            throws IOException, ServletException {

        if (exception.getCause() instanceof OAuth2IdConflictException){
            OAuth2IdConflictException OAuth2IdConflictException = (OAuth2IdConflictException) exception.getCause();
            response.getOutputStream().println(OAuth2IdConflictException.getMessage());
            response.setStatus(403);
        } else {
            super.onAuthenticationFailure(request, response, exception);
        }
    }

}
