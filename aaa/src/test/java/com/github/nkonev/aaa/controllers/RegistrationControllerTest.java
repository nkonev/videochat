package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.AbstractUtTestRunner;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.TestConstants;
import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.entity.redis.UserConfirmationToken;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.SecurityConfig;
import com.github.nkonev.aaa.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import jakarta.mail.Message;
import org.awaitility.Awaitility;
import org.eclipse.angus.mail.imap.IMAPMessage;
import org.hamcrest.Matchers;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;
import org.springframework.web.util.UriComponentsBuilder;

import java.net.URI;
import java.time.Duration;
import java.util.UUID;

import static com.github.nkonev.aaa.TestConstants.HEADER_SET_COOKIE;
import static com.github.nkonev.aaa.TestConstants.SESSION_COOKIE_NAME;
import static org.awaitility.Awaitility.await;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

//@DisplayName("Testing registration with GreenMail per test")
public class RegistrationControllerTest extends AbstractUtTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private CustomConfig customConfig;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationControllerTest.class);

    @BeforeAll
    public static void ba() {
        Awaitility.setDefaultTimeout(Duration.ofSeconds(30));
    }

    @Test
    public void testConfirmationSuccess() throws Exception {
        final String email = "newly@example.com";
        final String username = "newly";
        final String password = "password";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null,  null, password, email);

        // register
        MvcResult createAccountRequest = mockMvc.perform(
                MockMvcRequestBuilders.post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .queryParam("language", Language.en.name())
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
        String createAccountStr = createAccountRequest.getResponse().getContentAsString();
        LOGGER.info(createAccountStr);

        // we cannot login without confirmation
        mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                    .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                    .param(SecurityConfig.USERNAME_PARAMETER, username)
                    .param(SecurityConfig.PASSWORD_PARAMETER, password)
                    .with(csrf())
            )
            .andExpect(status().is4xxClientError())
            .andExpect(header().stringValues(HEADER_SET_COOKIE, Matchers.not(Matchers.hasItem(SESSION_COOKIE_NAME))));

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 1);
            IMAPMessage imapMessage = (IMAPMessage)messages[0];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            String tokenUuidString = UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0);
            Assertions.assertTrue(userConfirmationTokenRepository.existsById(tokenUuidString));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getRegistrationConfirmExitSuccessUrl()))
                // assert server returns session id
                .andExpect(cookie().value(SESSION_COOKIE_NAME, Matchers.notNullValue()))
            ;
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
                .andExpect(status().isOk())
                // assert server returns session id
                .andExpect(cookie().value(SESSION_COOKIE_NAME, Matchers.notNullValue()));


    }

    @Test
    public void testRegistrationConfirmationAfterReissuingTokenSuccess() throws Exception {
        final String email = "newbie@example.com";
        final String username = "newbie";
        final String password = "password";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null,  null, password, email);

        // register
        MvcResult createAccountRequest = mockMvc.perform(
                MockMvcRequestBuilders.post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                    .content(objectMapper.writeValueAsString(createUserDTO))
                    .queryParam("language", Language.en.name())
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
                    post(Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL + "?email=" + email + "&language=en")
                        .with(csrf())
                )
                .andExpect(status().isOk());
            Assertions.assertEquals(tokenCountBeforeResend+1, userConfirmationTokenRepository.count());
        }

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 2); // backend should send two email: a) during registration; b) during confirmation token reissue
            IMAPMessage imapMessage = (IMAPMessage)messages[1];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            String tokenUuidString = UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0);
            Assertions.assertTrue(userConfirmationTokenRepository.existsById(tokenUuidString));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getRegistrationConfirmExitSuccessUrl()))
            ;
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
                    post(Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL + "?email=" + email + "&language=en")
                        .with(csrf())
                )
                .andExpect(status().isOk());
            Assertions.assertEquals(tokenCountBeforeResend, userConfirmationTokenRepository.count());
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
    }

    @Test
    public void testRegistrationPasswordIsRequired() throws Exception {
        final String email = "newbie@example.com";
        final String username = "newbie";

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null, null, null, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .queryParam("language", Language.en.name())
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

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null, null, "123", email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .queryParam("language", Language.en.name())
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

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null, null, password, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .queryParam("language", Language.en.name())
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

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null, null, password, email);

        // register
        MvcResult createAccountResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.REGISTER)
                        .content(objectMapper.writeValueAsString(createUserDTO))
                        .queryParam("language", Language.en.name())
                        .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();
        String stringResponse = createAccountResult.getResponse().getContentAsString();
        LOGGER.info(stringResponse);

        UserAccount userAccountAfter = userAccountRepository.findByEmail(email).orElseThrow(() -> new RuntimeException("user account not found in test"));

        // check that initial user account is not affected
        Assertions.assertEquals(userAccountBefore.id(), userAccountAfter.id());
        Assertions.assertEquals(userAccountBefore.avatar(), userAccountAfter.avatar());
        Assertions.assertEquals(TestConstants.USER_ALICE, userAccountBefore.username());
        Assertions.assertEquals(userAccountBefore.username(), userAccountAfter.username());
        Assertions.assertEquals(userAccountBefore.password(), userAccountAfter.password());
        Assertions.assertEquals(userAccountBefore.role(), userAccountAfter.role());
    }


    @Test
    public void testConfirmationTokenNotFound() throws Exception {
        String token = UUID.randomUUID().toString(); // create random token
        userConfirmationTokenRepository.deleteById(token); // if random token exists we delete it

        // create /confirm?uuid=<uuid>
        String uri = UriComponentsBuilder.fromUriString(customConfig.getBaseUrl() + Constants.Urls.REGISTER_CONFIRM).queryParam(Constants.Urls.UUID, token).build().toUriString();

        mockMvc.perform(get(uri))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getRegistrationConfirmExitTokenNotFoundUrl()))
        ;
    }

    @Test
    public void testConfirmationUserNotFound() throws Exception {
        String tokenUuid = UUID.randomUUID().toString(); // create random token
        UserConfirmationToken token1 = new UserConfirmationToken(tokenUuid, -999L, 180);
        userConfirmationTokenRepository.save(token1); // save it

        // create /confirm?uuid=<uuid>
        String uri = UriComponentsBuilder.fromUriString(customConfig.getBaseUrl() + Constants.Urls.REGISTER_CONFIRM).queryParam(Constants.Urls.UUID, tokenUuid).build().toUriString();

        mockMvc.perform(get(uri))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getRegistrationConfirmExitUserNotFoundUrl()))
        ;

    }

    @Test
    public void testAttackerCannotStealLockedUserAccount() throws Exception {
        String bobEmail = "bob@example.com";
        UserAccount bob = userAccountRepository.findByEmail(bobEmail).orElseThrow(()->new RuntimeException("bob not found in test"));

        bob = bob.withLocked(true);
        bob = userAccountRepository.save(bob);

        // attacker
        long tokenCountBeforeResend = userConfirmationTokenRepository.count();
        mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CONFIRMATION_EMAIL+"?email="+bobEmail + "&language=en")
                    .with(csrf())
        )
                .andExpect(status().isOk());
        Assertions.assertEquals(tokenCountBeforeResend, userConfirmationTokenRepository.count(), "new token shouldn't appear when attacker attempts reactivate banned(locked) user");
    }


}
