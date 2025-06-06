package name.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import name.nkonev.aaa.AbstractMockMvcTestRunner;
import name.nkonev.aaa.TestConstants;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.*;
import name.nkonev.aaa.entity.jdbc.CreationType;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.repository.redis.ChangeEmailConfirmationTokenRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.services.EventReceiver;
import name.nkonev.aaa.tasks.SyncLdapTask;
import name.nkonev.aaa.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import jakarta.mail.Message;
import jakarta.servlet.http.Cookie;
import org.awaitility.Awaitility;
import org.eclipse.angus.mail.imap.IMAPMessage;
import org.hamcrest.CoreMatchers;
import org.hamcrest.Matchers;
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

import java.net.HttpCookie;
import java.net.URI;
import java.time.Duration;
import java.util.Arrays;
import java.util.Map;
import java.util.Optional;

import static name.nkonev.aaa.TestConstants.*;
import static org.awaitility.Awaitility.await;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("User profile")
public class UserProfileControllerTest extends AbstractMockMvcTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private EventReceiver receiver;

    @Autowired
    private ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    @Autowired
    private SyncLdapTask syncLdapTask;

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
    @Test
    public void testGetAliceProfileWhichNotContainsPassword() throws Exception {
        MvcResult getPostsRequest = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andExpect(jsonPath("$.expiresAt").exists())
                .andReturn();
    }

    private UserAccount getUserFromBd(String userName) {
        return userAccountRepository.findByLogin(userName).orElseThrow(() ->  new RuntimeException("User '" + userName + "' not found during test"));
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void fullyAuthenticatedUserCanChangeHerProfile() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String initialPassword = userAccount.password();

        final String newLogin = "new_alice";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
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
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.login").value(newLogin))
                .andExpect(jsonPath("$.password").doesNotExist())
                .andReturn();

        await().ignoreExceptions().until(() -> receiver.sizeChanged(), s -> s > 0);
        Assertions.assertEquals(1, receiver.sizeChanged());
        final var userAccountEvent = receiver.getLastChanged();
        Assertions.assertEquals(newLogin, userAccountEvent.login());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void fullyAuthenticatedUserCanChangeHerProfileAndPassword() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String initialPassword = userAccount.password();
        final String newLogin = "new_alice12";
        final String newPassword = "new_alice_password";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);
        edit = edit.withPassword(newPassword);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
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
    @Test
    @WithUserDetails(USER_BOB)
    public void fullyAuthenticatedUserCannotChangeForeignProfile() throws Exception {
        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_ALICE);
        String foreignUserAccountLogin = foreignUserAccount.login();
        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(foreignUserAccount);

        final String badLogin = "stolen";
        edit = edit.withLogin(badLogin);
        Map<String, Object> userMap = objectMapper.readValue(objectMapper.writeValueAsString(edit), new TypeReference<Map<String, Object>>(){} );
        userMap.put("id", foreignUserAccount.id());

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(userMap))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        UserAccount foreignPotentiallyAffectedUserAccount = getUserFromBd(TestConstants.USER_ALICE);
        Assertions.assertEquals(foreignUserAccountLogin, foreignPotentiallyAffectedUserAccount.login());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void fullyAuthenticatedUserCannotTakeForeignLogin() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);

        final String newLogin = USER_BOB;

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
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
                patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
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

    @Test
    public void userCanSeeTheirOwnEmail() throws Exception {
        String session = getMockMvcSession(TestConstants.USER_ADMIN, password);
        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_ADMIN);
        RequestEntity requestEntity = RequestEntity
            .get(new URI(urlWithContextPath() + Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE))
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
    @Test
    public void userCannotSeeAnybodyProfileEmail() throws Exception {
        UserAccount bob = getUserFromBd(USER_BOB);

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER+Constants.Urls.SEARCH)
                        .param("searchString", bob.login())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.items[0].email").doesNotExist())
                .andExpect(jsonPath("$.items[0].login").value(USER_BOB))
                .andReturn();

    }

    @Test
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

    @Test
    public void userCanSeeOnlyOwnProfileEmail() throws Exception {
        String session = getMockMvcSession(TestConstants.USER_ALICE, TestConstants.USER_ALICE_PASSWORD);
        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        UserAccount foreignUserAccount = getUserFromBd(USER_BOB);
        RequestEntity requestEntity = RequestEntity
            .get(new URI(urlWithContextPath() + Constants.Urls.EXTERNAL_API +Constants.Urls.USER + "/" + foreignUserAccount.id()))
            .header(TestConstants.HEADER_COOKIE, headerValue).build();
        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        var response = objectMapper.readValue(responseEntity.getBody(), JsonNode.class);
        Assertions.assertEquals(foreignUserAccount.id(), response.get("id").asLong());
        Assertions.assertNull(response.get("email"));
    }

    @Test
    public void userCannotManageSessions() throws Exception {
        String session = getMockMvcSession(TestConstants.USER_ALICE, TestConstants.USER_ALICE_PASSWORD);

        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        RequestEntity requestEntity = RequestEntity
                .get(new URI(urlWithContextPath() + Constants.Urls.EXTERNAL_API + Constants.Urls.SESSIONS + "?userId=1"))
                .header(TestConstants.HEADER_COOKIE, headerValue).build();

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        String str = responseEntity.getBody();

        Assertions.assertEquals(403, responseEntity.getStatusCodeValue());

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>() { });
        Assertions.assertEquals("Forbidden", resp.get("message"));
    }

    @Test
    public void adminCanManageSessions() throws Exception {
        String session = getMockMvcSession(username, password);

        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        RequestEntity requestEntity = RequestEntity
                .get(new URI(urlWithContextPath() + Constants.Urls.EXTERNAL_API + Constants.Urls.SESSIONS + "?userId=1"))
                .header(TestConstants.HEADER_COOKIE, headerValue).build();

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(200, responseEntity.getStatusCodeValue());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void userCannotManageSessionsView() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.SEARCH)
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.items[2].canDelete").value(false))
                .andExpect(jsonPath("$.items[2].canChangeRole").value(false))
                .andExpect(jsonPath("$.items[2].canLock").value(false))

                .andReturn();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void adminCanManageSessionsView() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.SEARCH)
            )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.items[2].canDelete").value(true))
                .andExpect(jsonPath("$.items[2].canChangeRole").value(true))
                .andExpect(jsonPath("$.items[2].canLock").value(true))

                .andReturn();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void adminCanLock() throws Exception {
        final long userId = 10;

        // lock user 10
        LockDTO lockDTO = new LockDTO(userId, true);
        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.LOCK)
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
    @Test
    public void userCanNotLock() throws Exception {
        final long userId = 10;

        // lock user 10
        LockDTO lockDTO = new LockDTO(userId, true);
        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.LOCK)
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

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void userSearchJohnSmithTrim() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                        .param("searchString", " John Smith")
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.items.length()").value(1))
                .andExpect(jsonPath("$.items[0].login").value("John Smith"))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);

    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void userSearchJohnSmithIgnoreCase() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                        .param("searchString", "john sMith")

        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.items.length()").value(1))
                .andExpect(jsonPath("$.items[0].login").value("John Smith"))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);

    }

    private long createUserForDelete(String login) {
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                login, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, login+"@example.com", null, null, null, null, null, null, null, null, null, null, null);
        userAccount = userAccountRepository.save(userAccount);

        return userAccount.id();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void adminCanDeleteUser() throws Exception {

        long id = createUserForDelete("lol2");

        MvcResult mvcResult = mockMvc.perform(
                delete(Constants.Urls.EXTERNAL_API + Constants.Urls.USER)
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

        final UserAccountEventDeletedDTO userAccountEvent = receiver.getLastDeleted();
        Assertions.assertEquals(id, userAccountEvent.userId());
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void userCannotDeleteUser() throws Exception {
        long id = createUserForDelete("lol1");

        MvcResult mvcResult = mockMvc.perform(
                delete(Constants.Urls.EXTERNAL_API + Constants.Urls.USER)
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
        String session = getMockMvcSession("admin", "admin");

        mockMvc.perform(
                        get(Constants.Urls.EXTERNAL_API +Constants.Urls.SESSIONS+"/my")
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
        getMockMvcSession(USER_BOB_LDAP, USER_BOB_LDAP_PASSWORD);
        Optional<UserAccount> bob = userAccountRepository.findByLogin(USER_BOB_LDAP);
        Assertions.assertTrue(bob.isPresent());
        var gotBob = bob.get();
        Assertions.assertEquals(USER_BOB_LDAP_ID, gotBob.ldapId());
        Map<String, Session> bobRedisSessions = aaaUserDetailsService.getSessions(USER_BOB_LDAP);
        Assertions.assertEquals(1, bobRedisSessions.size());
        Assertions.assertTrue(Arrays.asList(gotBob.roles()).contains(UserRole.ROLE_ADMIN));
        Assertions.assertEquals(USER_BOB_LDAP_EMAIL, gotBob.email());

        userAccountRepository.save(gotBob
                .withEmail("a@b.com")
                .withRoles(new UserRole[]{})
        );

        var overridedBob = userAccountRepository.findByLogin(USER_BOB_LDAP).get();
        Assertions.assertEquals("a@b.com", overridedBob.email());
        Assertions.assertEquals(0, overridedBob.roles().length);

        syncLdapTask.doWork();

        var restoredBob = userAccountRepository.findByLogin(USER_BOB_LDAP).get();
        Assertions.assertEquals(USER_BOB_LDAP_EMAIL, restoredBob.email());
        Assertions.assertTrue(Arrays.asList(restoredBob.roles()).contains(UserRole.ROLE_ADMIN));
        Assertions.assertTrue(restoredBob.syncLdapDateTime().isAfter(gotBob.syncLdapDateTime()));
    }

    @Test
    public void ldapSyncCreatesUsers() {
        var ldapUsersBefore = jdbcTemplate.queryForObject("select count (*) from user_account where ldap_id is not null", Long.class);
        Assertions.assertEquals(0L, ldapUsersBefore);

        syncLdapTask.doWork();

        var ldapUsersAfter = jdbcTemplate.queryForObject("select count (*) from user_account where ldap_id is not null", Long.class);
        Assertions.assertEquals(4L, ldapUsersAfter);
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
                MockMvcRequestBuilders.patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
                    .content(objectMapper.writeValueAsString(createUserDTO))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isOk())
            .andReturn();

        var user = userAccountRepository.findByLogin(username).get();
        Assertions.assertEquals(oldEmail, user.email());

        Assertions.assertEquals(email, getTokenNewEmail(user.id()));

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 1);
            IMAPMessage imapMessage = (IMAPMessage)messages[0];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            Assertions.assertTrue(changeEmailConfirmationTokenRepository.existsById(user.id()));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, aaaProperties.confirmChangeEmailExitSuccessUrl()))
            ;
            Assertions.assertFalse(changeEmailConfirmationTokenRepository.existsById(user.id()));

            var userAfterConfirm = userAccountRepository.findByLogin(username).get();
            Assertions.assertEquals(email, userAfterConfirm.email());
        }

    }

    private String getTokenNewEmail(long userId) {
        return changeEmailConfirmationTokenRepository.findById(userId).map(ChangeEmailConfirmationToken::newEmail).orElse("");
    }

    final String userForChangeEmail1 = "generated_user_21";
    @WithUserDetails(userForChangeEmail1)
    @Test
    public void testConfirmationOfChangingEmailAfterReissuingTokenSuccess() throws Exception {
        final String oldEmail = "generated21@example.com";
        final String email = "generated_user_21_changed@example.com";
        final String username = userForChangeEmail1;

        EditUserDTO createUserDTO = new EditUserDTO(username, null, null,  null, null, email);

        long tokenCountBeforeSend = changeEmailConfirmationTokenRepository.count();

        // changeEmail
        mockMvc.perform(
                MockMvcRequestBuilders.patch(Constants.Urls.EXTERNAL_API + Constants.Urls.PROFILE)
                    .content(objectMapper.writeValueAsString(createUserDTO))
                    .contentType(MediaType.APPLICATION_JSON_UTF8_VALUE)
                    .with(csrf())
            )
            .andExpect(status().isOk())
            .andReturn();

        var user = userAccountRepository.findByLogin(username).get();
        Assertions.assertEquals(oldEmail, user.email());
        Assertions.assertEquals(email, getTokenNewEmail(user.id()));

        var firstToken = changeEmailConfirmationTokenRepository.findById(user.id());
        long tokenCountBeforeResend = changeEmailConfirmationTokenRepository.count();
        Assertions.assertEquals(tokenCountBeforeSend+1, tokenCountBeforeResend);

        // just retrieve first email due to peculiarities of greenmail
        try (Retriever r = new Retriever(greenMail.getImap())) {
            await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 1);
        }

        // user lost email and reissues token
        {
            mockMvc.perform(
                    post(Constants.Urls.EXTERNAL_API + Constants.Urls.RESEND_CHANGE_EMAIL_CONFIRM)
                        .with(csrf())
                )
                .andExpect(status().isOk());

            // we override old token so count is the same
            Assertions.assertEquals(tokenCountBeforeResend, changeEmailConfirmationTokenRepository.count());
            var secondToken = changeEmailConfirmationTokenRepository.findById(user.id());
            Assertions.assertNotEquals(firstToken.get().uuid(), secondToken.get().uuid());
        }

        // confirm
        // http://www.icegreen.com/greenmail/javadocs/com/icegreen/greenmail/util/Retriever.html
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = await().ignoreExceptions().until(() -> r.getMessages(email), msgs -> msgs.length == 2); // backend should send two email: a) during the first attempt; b) during the second attempt
            IMAPMessage imapMessage = (IMAPMessage)messages[1]; // get the second email
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            Assertions.assertTrue(changeEmailConfirmationTokenRepository.existsById(user.id()));

            // perform confirm
            mockMvc.perform(get(parsedUrl))
                .andExpect(status().is3xxRedirection())
                .andExpect(header().string(HttpHeaders.LOCATION, aaaProperties.confirmChangeEmailExitSuccessUrl()))
            ;
            Assertions.assertFalse(changeEmailConfirmationTokenRepository.existsById(user.id()));

            var userAfterConfirm = userAccountRepository.findByLogin(username).get();
            Assertions.assertEquals(email, userAfterConfirm.email());
        }

    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void noErrorInCaseTooBigRequestedStartingFromItemId() throws Exception {
        MvcResult getPostRequest = mockMvc.perform(
                get(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.SEARCH)
                        .param("startingFromItemId", "10000000")
                        .param("size", "40")
                        .param("reverse", "false")
                        .param("includeStartingFrom", "true")

            )
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.items[0].login").value("forgot-password-user"))
            .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @Test
    public void adminCanAccessToJaeger() throws Exception {
        mockMvc.perform(
                get(Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE, Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH)
                    .header("x-forwarded-uri", "/jaeger")
            )
            .andDo(result -> {
                LOGGER.info(result.getResponse().getContentAsString());
            })
            .andExpect(status().isOk())
            ;
    }

    @WithUserDetails(USER_ALICE)
    @Test
    public void userCannotAccessToJaeger() throws Exception {

        mockMvc.perform(
                get(Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE, Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH)
                    .header("x-forwarded-uri", "/jaeger")
            )
            .andDo(result -> {
                LOGGER.info(result.getResponse().getContentAsString());
            })
            .andExpect(status().isForbidden())
            .andReturn();
    }

}
