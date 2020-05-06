package com.github.nkonev.blog.controllers;

import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.dto.EditUserDTO;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.entity.redis.UserConfirmationToken;
import com.github.nkonev.blog.extensions.GreenMailExtension;
import com.github.nkonev.blog.extensions.GreenMailExtensionFactory;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.repository.redis.PasswordResetTokenRepository;
import com.github.nkonev.blog.security.SecurityConfig;
import com.github.nkonev.blog.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import com.sun.mail.imap.IMAPMessage;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;
import org.springframework.web.util.UriComponentsBuilder;

import javax.mail.Message;
import java.net.URI;
import java.util.UUID;

import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

//@DisplayName("Testing registration with GreenMail per test")
public class RegistrationControllerTest extends AbstractUtTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordResetTokenRepository passwordResetTokenRepository;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationControllerTest.class);

    @RegisterExtension
    protected GreenMailExtension greenMail = GreenMailExtensionFactory.build();

    @Test
    public void testConfirmationSuccess() throws Exception {
        final String email = "newbie@example.com";
        final String username = "newbie";
        final String password = "password";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, password, email);

        // register
        MvcResult createAccountRequest = mockMvc.perform(
                MockMvcRequestBuilders.post(Constants.Urls.API+ Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
        String createAccountStr = createAccountRequest.getResponse().getContentAsString();
        LOGGER.info(createAccountStr);

        // login unconfirmed fail
        mockMvc.perform(
                MockMvcRequestBuilders.post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(SecurityConfig.USERNAME_PARAMETER, username)
                        .param(SecurityConfig.PASSWORD_PARAMETER, password)
                        .with(csrf())
        )
                .andExpect(status().isUnauthorized());

        // user lost email and reissues token
        {
            long tokenCountBeforeResend = userConfirmationTokenRepository.count();
            mockMvc.perform(
                    post(Constants.Urls.API + Constants.Urls.RESEND_CONFIRMATION_EMAIL + "?email=" + email)
                            .with(csrf())
            )
                    .andExpect(status().isOk());
            Assertions.assertEquals(tokenCountBeforeResend+1, userConfirmationTokenRepository.count());
        }

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = r.getMessages(email);
            Assertions.assertEquals(2, messages.length, "backend should sent two email: a) during registration; b) during confirmation token reissue");
            IMAPMessage imapMessage = (IMAPMessage)messages[1];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            String tokenUuidString = UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0);
            Assertions.assertTrue(userConfirmationTokenRepository.existsById(tokenUuidString));

            // perform confirm
            mockMvc.perform(get(parsedUrl)).andExpect(status().isOk());
            Assertions.assertFalse(userConfirmationTokenRepository.existsById(tokenUuidString));
        }

        // login confirmed ok
        mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(SecurityConfig.USERNAME_PARAMETER, username)
                        .param(SecurityConfig.PASSWORD_PARAMETER, password)
                        .with(csrf())
        )
                .andExpect(status().isOk());

        // resend for already confirmed does nothing
        {
            long tokenCountBeforeResend = userConfirmationTokenRepository.count();
            mockMvc.perform(
                    post(Constants.Urls.API + Constants.Urls.RESEND_CONFIRMATION_EMAIL + "?email=" + email)
                            .with(csrf())
            )
                    .andExpect(status().isOk());
            Assertions.assertEquals(tokenCountBeforeResend, userConfirmationTokenRepository.count());
        }
    }

    @Test
    public void testRegistrationPasswordIsRequired() throws Exception {
        final String email = "newbie@example.com";
        final String username = "newbie";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("validation error"))
                .andExpect(jsonPath("$.message").value("password must be set"))
                .andReturn();
        String stringResponse = createAccountResult.getResponse().getContentAsString();
        LOGGER.info(stringResponse);

    }

    @Test
    public void testRegistrationPasswordNotEnoughLong() throws Exception {
        final String email = "newbie@example.com";
        final String username = "newbie";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, "123", email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("validation error"))
                .andExpect(jsonPath("$.message").value("password don't match requirements"))
                .andReturn();
        String stringResponse = createAccountResult.getResponse().getContentAsString();
        LOGGER.info(stringResponse);
    }

    @Test
    public void testRegistrationUserWithSameLoginAlreadyPresent() throws Exception {
        final String email = "newbie@example.com";
        final String username = TestConstants.USER_ALICE;
        final String password = "password";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, password, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("user already present"))
                .andExpect(jsonPath("$.message").value("User with login 'alice' is already present"))
                .andReturn();
        String stringResponse = createAccountResult.getResponse().getContentAsString();
        LOGGER.info(stringResponse);
    }

    @Test
    public void testRegistrationUserWithSameEmailAlreadyPresent() throws Exception {
        final String email = "alice@example.com";
        final String username = "newbie";
        final String password = "password";

        UserAccount userAccountBefore = userAccountRepository.findByEmail(email).orElseThrow(() -> new RuntimeException("user account not found in test"));

        EditUserDTO createUserDTO = new EditUserDTO(username, null, password, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
        String stringResponse = createAccountResult.getResponse().getContentAsString();
        LOGGER.info(stringResponse);

        UserAccount userAccountAfter = userAccountRepository.findByEmail(email).orElseThrow(() -> new RuntimeException("user account not found in test"));

        // check that initial user account is not affected
        Assertions.assertEquals(userAccountBefore.getId(), userAccountAfter.getId());
        Assertions.assertEquals(userAccountBefore.getAvatar(), userAccountAfter.getAvatar());
        Assertions.assertEquals(TestConstants.USER_ALICE, userAccountBefore.getUsername());
        Assertions.assertEquals(userAccountBefore.getUsername(), userAccountAfter.getUsername());
        Assertions.assertEquals(userAccountBefore.getPassword(), userAccountAfter.getPassword());
        Assertions.assertEquals(userAccountBefore.getRole(), userAccountAfter.getRole());
    }


    @Test
    public void testConfirmationTokenNotFound() throws Exception {
        String token = UUID.randomUUID().toString(); // create random token
        userConfirmationTokenRepository.deleteById(token); // if random token exists we delete it

        // create /confirm?uuid=<uuid>
        String uri = UriComponentsBuilder.fromUriString(Constants.Urls.CONFIRM).queryParam(Constants.Urls.UUID, token).build().toUriString();

        mockMvc.perform(get(uri))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, "/confirm/registration/token-not-found"))
        ;
    }

    @Test
    public void testConfirmationUserNotFound() throws Exception {
        String tokenUuid = UUID.randomUUID().toString(); // create random token
        UserConfirmationToken token1 = new UserConfirmationToken(tokenUuid, -999L, 180);
        userConfirmationTokenRepository.save(token1); // save it

        // create /confirm?uuid=<uuid>
        String uri = UriComponentsBuilder.fromUriString(Constants.Urls.CONFIRM).queryParam(Constants.Urls.UUID, tokenUuid).build().toUriString();

        mockMvc.perform(get(uri))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, "/confirm/registration/user-not-found"))
        ;

    }

    @Test
    public void testAttackerCannotStealLockedUserAccount() throws Exception {
        String bobEmail = "bob@example.com";
        UserAccount bob = userAccountRepository.findByEmail(bobEmail).orElseThrow(()->new RuntimeException("bob not found in test"));

        bob.setLocked(true);
        bob = userAccountRepository.save(bob);

        // attacker
        long tokenCountBeforeResend = userConfirmationTokenRepository.count();
        mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.RESEND_CONFIRMATION_EMAIL+"?email="+bobEmail)
                    .with(csrf())
        )
                .andExpect(status().isOk());
        Assertions.assertEquals(tokenCountBeforeResend, userConfirmationTokenRepository.count(), "new token shouldn't appear when attacker attempts reactivate banned(locked) user");
    }

    // scheme simplified, suspect that user's email doesn't stolen
    @Test
    public void userCanRequestPasswordOnlyOnOwnEmail() throws Exception {
        final String user = TestConstants.USER_BOB;
        final String email = user+"@example.com";
        final String newPassword = "new-password";

        // invoke resend, this sends url /password-reset?uuid=<uuid> and confirm code to email
        mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.REQUEST_PASSWORD_RESET+"?email="+email)
                        .with(csrf())
        )
                .andExpect(status().isOk());


        String passwordResetTokenUuidString;
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = r.getMessages(email);
            Assertions.assertEquals(1, messages.length, "backend should sent one email for password reset");
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
                post(Constants.Urls.API+ Constants.Urls.PASSWORD_RESET_SET_NEW)
                        .content(objectMapper.writeValueAsString(passwordResetDto))
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        ).andExpect(status().isOk());


        // ... this is changes his password
        // login with new password ok
        mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(SecurityConfig.USERNAME_PARAMETER, user)
                        .param(SecurityConfig.PASSWORD_PARAMETER, newPassword)
                        .with(csrf())
        )
                .andExpect(status().isOk());


    }

    @Test
    public void handlePasswordResetTokenNotFound() throws Exception {
        UUID tokenUuid = UUID.randomUUID();
        if (passwordResetTokenRepository.existsById(tokenUuid)) {
            passwordResetTokenRepository.deleteById(tokenUuid); // delete random if one is occasionally present
        }

        PasswordResetController.PasswordResetDto passwordResetDto = new PasswordResetController.PasswordResetDto(tokenUuid, "qwqwqwqwqwqwqwqw");

        mockMvc.perform(
                post(Constants.Urls.API+ Constants.Urls.PASSWORD_RESET_SET_NEW)
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
                post(Constants.Urls.API+ Constants.Urls.PASSWORD_RESET_SET_NEW)
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
                post(Constants.Urls.API+ Constants.Urls.PASSWORD_RESET_SET_NEW)
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
