package com.github.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.github.nkonev.aaa.AbstractUtTestRunner;
import com.github.nkonev.aaa.TestConstants;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.*;
import com.github.nkonev.aaa.entity.jdbc.CreationType;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaUserDetailsService;
import com.github.nkonev.aaa.security.SecurityConfig;
import com.github.nkonev.aaa.services.EventReceiver;
import com.github.nkonev.aaa.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import jakarta.mail.Message;
import jakarta.servlet.http.Cookie;
import org.awaitility.Awaitility;
import org.eclipse.angus.mail.imap.IMAPMessage;
import org.hamcrest.CoreMatchers;
import org.junit.jupiter.api.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.session.Session;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders;
import org.springframework.web.util.UriComponentsBuilder;

import java.net.HttpCookie;
import java.net.URI;
import java.time.Duration;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import static com.github.nkonev.aaa.TestConstants.*;
import static org.awaitility.Awaitility.await;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("User profile")
public class UserProfileControllerTest extends AbstractUtTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private EventReceiver receiver;

    @BeforeAll
    public static void ba() {
        Awaitility.setDefaultTimeout(Duration.ofSeconds(30));
    }

    @BeforeEach
    public void be() {
        receiver.clearChanged();
        receiver.clearDeleted();
    }

    @AfterEach
    public void ae() {
        receiver.clearChanged();
        receiver.clearDeleted();
    }

    private static final Logger LOGGER = LoggerFactory.getLogger(UserProfileControllerTest.class);


    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void testGetAliceProfileWhichNotContainsPassword() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andExpect(jsonPath("$.expiresAt").exists())
                .andReturn();
    }

    private UserAccount getUserFromBd(String userName) {
        return userAccountRepository.findByUsername(userName).orElseThrow(() ->  new RuntimeException("User '" + userName + "' not found during test"));
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void fullyAuthenticatedUserCanChangeHerProfile() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String initialPassword = userAccount.password();

        final String newLogin = "new_alice";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(newLogin))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        Assertions.assertEquals(initialPassword, getUserFromBd(newLogin).password(), "password shouldn't be affected if there isn't set explicitly");

        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(newLogin))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn();

        await().ignoreExceptions().until(() -> receiver.sizeChanged(), s -> s > 0);
        Assertions.assertEquals(1, receiver.sizeChanged());
        final UserAccountDTO userAccountEvent = receiver.getLastChanged();
        Assertions.assertEquals(newLogin, userAccountEvent.login());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void fullyAuthenticatedUserCanChangeHerProfileAndPassword() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String initialPassword = userAccount.password();
        final String newLogin = "new_alice12";
        final String newPassword = "new_alice_password";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);
        edit = edit.withPassword(newPassword);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(newLogin))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        UserAccount afterChange = getUserFromBd(newLogin);
        Assertions.assertNotEquals(initialPassword, afterChange.password(), "password should be changed if there is set explicitly");
        Assertions.assertTrue( passwordEncoder.matches(newPassword, afterChange.password()), "password should be changed if there is set explicitly");
    }


    /**
     * Bob wants steal Alice's account by rewrite login and set her id
     * @throws Exception
     */
    @org.junit.jupiter.api.Test
    @WithUserDetails(USER_BOB)
    public void fullyAuthenticatedUserCannotChangeForeignProfile() throws Exception {
        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_ALICE);
        String foreignUserAccountLogin = foreignUserAccount.username();
        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(foreignUserAccount);

        final String badLogin = "stolen";
        edit = edit.withLogin(badLogin);
        Map<String, Object> userMap = objectMapper.readValue(objectMapper.writeValueAsString(edit), new TypeReference<Map<String, Object>>(){} );
        userMap.put("id", foreignUserAccount.id());

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(userMap))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        UserAccount foreignPotentiallyAffectedUserAccount = getUserFromBd(TestConstants.USER_ALICE);
        Assertions.assertEquals(foreignUserAccountLogin, foreignPotentiallyAffectedUserAccount.username());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void fullyAuthenticatedUserCannotTakeForeignLogin() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);

        final String newLogin = USER_BOB;

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("user already present"))
                .andExpect(jsonPath("$.message").value("User with login 'bob' is already present"))
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void fullyAuthenticatedUserCannotTakeForeignEmail() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);

        final String newEmail = USER_BOB+"@example.com";
        final Optional<UserAccount> foreignBobAccountOptional = userAccountRepository.findByEmail(newEmail);
        final UserAccount foreignBobAccount = foreignBobAccountOptional.orElseThrow(()->new RuntimeException("foreign email '"+newEmail+"' must be present"));
        final long foreingId = foreignBobAccount.id();
        final String foreignPassword = foreignBobAccount.password();
        final String foreignEmail = foreignBobAccount.email();

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withEmail(newEmail);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andExpect(status().isOk()) // we care for emails
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        UserAccount foreignAccountAfter = getUserFromBd(USER_BOB);
        Assertions.assertEquals(foreingId, foreignAccountAfter.id().longValue());
        Assertions.assertEquals(foreignEmail, foreignAccountAfter.email());
        Assertions.assertEquals(foreignPassword, foreignAccountAfter.password());

    }

    @org.junit.jupiter.api.Test
    public void userCanSeeTheirOwnEmail() throws Exception {
        String session = getSession(TestConstants.USER_ADMIN, password);
        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_ADMIN);
        RequestEntity requestEntity = RequestEntity
            .get(new URI(urlWithContextPath() + Constants.Urls.PUBLIC_API +Constants.Urls.USER + "/" + foreignUserAccount.id()))
            .header(TestConstants.HEADER_COOKIE, headerValue).build();
        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        var response = objectMapper.readValue(responseEntity.getBody(), JsonNode.class);
        Assertions.assertEquals(foreignUserAccount.id(), response.get("id").asLong());
        Assertions.assertEquals(foreignUserAccount.email(), response.get("email").asText());
    }

    /**
     * Alice see Bob's profile, and she doesn't see his email
     * @throws Exception
     */
    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void userCannotSeeAnybodyProfileEmail() throws Exception {
        UserAccount bob = getUserFromBd(USER_BOB);
        String bobEmail = bob.email();

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER+Constants.Urls.SEARCH+"?userId="+bob.id())
                    .content(objectMapper.writeValueAsString(new UserProfileController.SearchUsersRequestDto(0, 0, false, false, bob.username())))
                    .contentType(MediaType.APPLICATION_JSON)
                    .with(csrf())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].email").doesNotExist())
                .andExpect(jsonPath("$[0].login").value(USER_BOB))
                .andExpect(content().string(CoreMatchers.not(CoreMatchers.containsString(bobEmail))))
                .andReturn();

    }

    @org.junit.jupiter.api.Test
    public void testGetManyUsersInternal() throws Exception {
        UserAccount bob = getUserFromBd(USER_BOB);
        UserAccount alice = getUserFromBd(TestConstants.USER_ALICE);

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.INTERNAL_API + Constants.Urls.USER+Constants.Urls.LIST+"?userId="+bob.id()+"&userId="+alice.id())
            )
            .andExpect(status().isOk())
            .andExpect(jsonPath("$[0].login").value(TestConstants.USER_ALICE))
            .andExpect(jsonPath("$[1].login").value(USER_BOB))
            .andReturn();

    }

    @org.junit.jupiter.api.Test
    public void userCanSeeOnlyOwnProfileEmail() throws Exception {
        String session = getSession(TestConstants.USER_ALICE, TestConstants.USER_ALICE_PASSWORD);
        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        UserAccount foreignUserAccount = getUserFromBd(USER_BOB);
        RequestEntity requestEntity = RequestEntity
            .get(new URI(urlWithContextPath() + Constants.Urls.PUBLIC_API +Constants.Urls.USER + "/" + foreignUserAccount.id()))
            .header(TestConstants.HEADER_COOKIE, headerValue).build();
        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        var response = objectMapper.readValue(responseEntity.getBody(), JsonNode.class);
        Assertions.assertEquals(foreignUserAccount.id(), response.get("id").asLong());
        Assertions.assertNull(response.get("email"));
    }

    @org.junit.jupiter.api.Test
    public void userCannotManageSessions() throws Exception {
        String session = getSession(TestConstants.USER_ALICE, TestConstants.USER_ALICE_PASSWORD);

        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        RequestEntity requestEntity = RequestEntity
                .get(new URI(urlWithContextPath() + Constants.Urls.PUBLIC_API + Constants.Urls.SESSIONS + "?userId=1"))
                .header(TestConstants.HEADER_COOKIE, headerValue).build();

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        String str = responseEntity.getBody();

        Assertions.assertEquals(403, responseEntity.getStatusCodeValue());

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>() { });
        Assertions.assertEquals("Forbidden", resp.get("message"));
    }

    @org.junit.jupiter.api.Test
    public void adminCanManageSessions() throws Exception {
        String session = getSession(username, password);

        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        RequestEntity requestEntity = RequestEntity
                .get(new URI(urlWithContextPath() + Constants.Urls.PUBLIC_API + Constants.Urls.SESSIONS + "?userId=1"))
                .header(TestConstants.HEADER_COOKIE, headerValue).build();

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(200, responseEntity.getStatusCodeValue());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void userCannotManageSessionsView() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                    .content("{}")
                    .contentType(MediaType.APPLICATION_JSON_UTF8)
                    .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[2].canDelete").value(false))
                .andExpect(jsonPath("$[2].canChangeRole").value(false))
                .andExpect(jsonPath("$[2].canLock").value(false))

                .andReturn();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @org.junit.jupiter.api.Test
    public void adminCanManageSessionsView() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                    .content("{}")
                    .contentType(MediaType.APPLICATION_JSON_UTF8)
                    .with(csrf())
            )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[2].canDelete").value(true))
                .andExpect(jsonPath("$[2].canChangeRole").value(true))
                .andExpect(jsonPath("$[2].canLock").value(true))

                .andReturn();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @org.junit.jupiter.api.Test
    public void adminCanLock() throws Exception {
        final long userId = 10;

        // lock user 10
        LockDTO lockDTO = new LockDTO(userId, true);
        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.LOCK)
                        .content(objectMapper.writeValueAsBytes(lockDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andReturn();

        // check that user 10 is locked
        UserAccount userAccountFound = userAccountRepository.findById(userId).orElseThrow(() -> new RuntimeException("User not found"));
        Assertions.assertTrue(userAccountFound.locked());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void userCanNotLock() throws Exception {
        final long userId = 10;

        // lock user 10
        LockDTO lockDTO = new LockDTO(userId, true);
        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.LOCK)
                        .content(objectMapper.writeValueAsBytes(lockDTO))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isForbidden())
                .andReturn();
    }

    @org.junit.jupiter.api.Test
    public void userSearchJohnSmithTrim() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                        .content("""
                                {
                                    "searchString": "%s"
                                }""".formatted(" John Smith"))
                        .with(csrf())
                        .contentType(MediaType.APPLICATION_JSON)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$.[0].login").value("John Smith"))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);

    }

    @org.junit.jupiter.api.Test
    public void userSearchJohnSmithIgnoreCase() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                .content("""
                        {
                            "searchString": "%s"
                        }""".formatted("john sMith"))
                .with(csrf())
                .contentType(MediaType.APPLICATION_JSON)

        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$.[0].login").value("John Smith"))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);

    }

    private long createUserForDelete(String login) {
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                login, null, null, null, null,false, false, true, true,
                UserRole.ROLE_USER, login+"@example.com", null, null, null, null, null, null, null);
        userAccount = userAccountRepository.save(userAccount);

        return userAccount.id();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void adminCanDeleteUser() throws Exception {

        long id = createUserForDelete("lol2");

        MvcResult mvcResult = mockMvc.perform(
                delete(Constants.Urls.PUBLIC_API + Constants.Urls.USER)
                        .param("userId", ""+id)
                        .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andReturn();

        await().ignoreExceptions().until(() -> receiver.sizeDeleted(), s -> s > 0);
        Assertions.assertEquals(1, receiver.sizeDeleted());

        final UserAccountDeletedEventDTO userAccountEvent = receiver.getLastDeleted();
        Assertions.assertEquals(id, userAccountEvent.userId());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void userCannotDeleteUser() throws Exception {
        long id = createUserForDelete("lol1");

        MvcResult mvcResult = mockMvc.perform(
                delete(Constants.Urls.PUBLIC_API + Constants.Urls.USER)
                .param("userId", ""+id)
                        .with(csrf())
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isForbidden())
                .andReturn();
    }

    @Test
    public void testMySessions() throws Exception {
        String session = getSession("admin", "admin");

        mockMvc.perform(
                        get(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS+"/my")
                                .cookie(new Cookie(getAuthCookieName(), session))
                ).andDo(mvcResult1 -> {
                    LOGGER.info(mvcResult1.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andReturn();
    }

    @Test
    public void ldapLoginTest() throws Exception {
        // https://spring.io/guides/gs/authenticating-ldap/
        getSession(USER_BOB_LDAP, USER_BOB_LDAP_PASSWORD);
        Optional<UserAccount> bob = userAccountRepository.findByUsername(USER_BOB_LDAP);
        Assertions.assertTrue(bob.isPresent());
        Assertions.assertEquals(USER_BOB_LDAP_ID, bob.get().ldapId());
        Map<String, Session> bobRedisSessions = aaaUserDetailsService.getSessions(USER_BOB_LDAP);
        Assertions.assertEquals(1, bobRedisSessions.size());
    }

    final String userForChangeEmail0 = "generated_user_20";
    @WithUserDetails(userForChangeEmail0)
    @Test
    public void testConfirmationOfChangingEmailSuccess() throws Exception {
        final String oldEmail = "generated20@example.com";
        final String email = "generated_user_20_changed@example.com";
        final String username = userForChangeEmail0;

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null,  null, null, email);

        // changeEmail
        mockMvc.perform(
                MockMvcRequestBuilders.patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                    .content(objectMapper.writeValueAsString(createUserDTO))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isOk())
            .andReturn();

        var user = userAccountRepository.findByUsername(username).get();
        Assertions.assertEquals(oldEmail, user.email());
        Assertions.assertEquals(email, user.newEmail());

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 1);
            IMAPMessage imapMessage = (IMAPMessage)messages[0];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            var tokenUuid = UUID.fromString(UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0));
            Assertions.assertTrue(changeEmailConfirmationTokenRepository.existsById(tokenUuid));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getConfirmChangeEmailExitSuccessUrl()))
            ;
            Assertions.assertFalse(changeEmailConfirmationTokenRepository.existsById(tokenUuid));

            var userAfterConfirm = userAccountRepository.findByUsername(username).get();
            Assertions.assertEquals(email, userAfterConfirm.email());
            Assertions.assertNull( userAfterConfirm.newEmail());
        }

    }

    final String userForChangeEmail1 = "generated_user_21";
    @WithUserDetails(userForChangeEmail1)
    @Test
    public void testConfirmationOfChangingEmailAfterReissuingTokenSuccess() throws Exception {
        final String oldEmail = "generated21@example.com";
        final String email = "generated_user_21_changed@example.com";
        final String username = userForChangeEmail1;

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null,  null, null, email);

        // changeEmail
        mockMvc.perform(
                MockMvcRequestBuilders.patch(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                    .content(objectMapper.writeValueAsString(createUserDTO))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isOk())
            .andReturn();

        var user = userAccountRepository.findByUsername(username).get();
        Assertions.assertEquals(oldEmail, user.email());
        Assertions.assertEquals(email, user.newEmail());

        // user lost email and reissues token
        {
            long tokenCountBeforeResend = changeEmailConfirmationTokenRepository.count();
            mockMvc.perform(
                    post(Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CHANGE_EMAIL_CONFIRM)
                        .with(csrf())
                )
                .andExpect(status().isOk());
            Assertions.assertEquals(tokenCountBeforeResend+1, changeEmailConfirmationTokenRepository.count());
        }

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 2); // backend should send two email: a) during the first attempt; b) during the second attempt
            IMAPMessage imapMessage = (IMAPMessage)messages[1];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            var tokenUuid = UUID.fromString(UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0));
            Assertions.assertTrue(changeEmailConfirmationTokenRepository.existsById(tokenUuid));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, customConfig.getConfirmChangeEmailExitSuccessUrl()))
            ;
            Assertions.assertFalse(changeEmailConfirmationTokenRepository.existsById(tokenUuid));

            var userAfterConfirm = userAccountRepository.findByUsername(username).get();
            Assertions.assertEquals(email, userAfterConfirm.email());
            Assertions.assertNull( userAfterConfirm.newEmail());
        }

    }

}
