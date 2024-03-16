package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.AbstractUtTestRunner;
import com.github.nkonev.aaa.TestConstants;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.repository.redis.PasswordResetTokenRepository;
import com.github.nkonev.aaa.security.SecurityConfig;
import com.github.nkonev.aaa.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import jakarta.mail.Message;
import org.eclipse.angus.mail.imap.IMAPMessage;
import org.hamcrest.Matchers;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.web.util.UriComponentsBuilder;

import java.net.URI;
import java.util.UUID;

import static com.github.nkonev.aaa.TestConstants.SESSION_COOKIE_NAME;
import static org.awaitility.Awaitility.await;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("Password reset")
public class PasswordResetControllerTest extends AbstractUtTestRunner {

    @Autowired
    private PasswordResetTokenRepository passwordResetTokenRepository;


    // presume that user's email doesn't stolen
    @Test
    public void passwordReset() throws Exception {
        final String user = TestConstants.USER_BOB;
        final String email = user+"@example.com";
        final String newPassword = "new-password";

        // invoke resend, this sends url /password-reset?uuid=<uuid> and confirm code to email
        mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.REQUEST_PASSWORD_RESET)
                    .queryParam("email", email)
                    .with(csrf())
            )
            .andExpect(status().isOk());


        String passwordResetTokenUuidString;
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 1);
            IMAPMessage imapMessage = (IMAPMessage)messages[0];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            passwordResetTokenUuidString = UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0);
        }

        // after open link user see "input new password dialog"
        // user inputs code, code compares with another in ResetPasswordToken
        PasswordResetController.PasswordResetDto passwordResetDto = new PasswordResetController.PasswordResetDto(UUID.fromString(passwordResetTokenUuidString), newPassword);

        // user click "set new password" button in modal
        mockMvc.perform(
            post(Constants.Urls.PUBLIC_API + Constants.Urls.PASSWORD_RESET_SET_NEW)
                .content(objectMapper.writeValueAsString(passwordResetDto))
                .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                .with(csrf())
        )
            .andExpect(status().isOk())
            // assert server returns session id
            .andExpect(cookie().value(SESSION_COOKIE_NAME, Matchers.notNullValue()));


        // ... this is changes his password
        // login with new password ok
        mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                    .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                    .param(SecurityConfig.USERNAME_PARAMETER, user)
                    .param(SecurityConfig.PASSWORD_PARAMETER, newPassword)
                    .with(csrf())
            )
            .andExpect(status().isOk())
            // assert server returns session id
            .andExpect(cookie().value(SESSION_COOKIE_NAME, Matchers.notNullValue()));


    }

    @Test
    public void handlePasswordResetTokenNotFound() throws Exception {
        UUID tokenUuid = UUID.randomUUID();
        if (passwordResetTokenRepository.existsById(tokenUuid)) {
            passwordResetTokenRepository.deleteById(tokenUuid); // delete random if one is occasionally present
        }

        PasswordResetController.PasswordResetDto passwordResetDto = new PasswordResetController.PasswordResetDto(tokenUuid, "qwqwqwqwqwqwqwqw");

        mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PASSWORD_RESET_SET_NEW)
                    .content(objectMapper.writeValueAsString(passwordResetDto))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isForbidden())
            .andExpect(jsonPath("$.message").value("password reset token not found or expired"))
            .andExpect(jsonPath("$.error").value("password reset"))
        ;
    }

    @Test
    public void handlePasswordResetTokenExpired() throws Exception {
        UUID tokenUuid = UUID.randomUUID();
        if (passwordResetTokenRepository.existsById(tokenUuid)) {
            passwordResetTokenRepository.deleteById(tokenUuid); // delete random if one is occasionally present
        }

        PasswordResetController.PasswordResetDto passwordResetDto = new PasswordResetController.PasswordResetDto(tokenUuid, "qwqwqwqwqwqwqwqw");

        mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PASSWORD_RESET_SET_NEW)
                    .content(objectMapper.writeValueAsString(passwordResetDto))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isForbidden())
            .andExpect(jsonPath("$.message").value("password reset token not found or expired"))
            .andExpect(jsonPath("$.error").value("password reset"))
        ;
    }

    @org.junit.jupiter.api.Test
    public void resetPasswordSetNewPasswordValidation() throws Exception {
        String emptyPassword = null;
        PasswordResetController.PasswordResetDto passwordResetDto = new PasswordResetController.PasswordResetDto(UUID.randomUUID(), emptyPassword);

        mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PASSWORD_RESET_SET_NEW)
                    .content(objectMapper.writeValueAsString(passwordResetDto))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.error").value("validation error"))
            .andExpect(jsonPath("$.message").value("validation error, see validationErrors[]"))
            .andExpect(jsonPath("$.validationErrors[0].field").value("newPassword"))
            .andExpect(jsonPath("$.validationErrors[0].message").value("must not be empty"))
        ;

    }
}
