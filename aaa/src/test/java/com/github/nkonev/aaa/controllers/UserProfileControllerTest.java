package com.github.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.github.nkonev.aaa.AbstractUtTestRunner;
import com.github.nkonev.aaa.TestConstants;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.dto.LockDTO;
import com.github.nkonev.aaa.dto.UserAccountDTO;
import com.github.nkonev.aaa.entity.jdbc.CreationType;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaUserDetailsService;
import com.github.nkonev.aaa.services.EventReceiver;
import com.google.common.util.concurrent.Uninterruptibles;
import org.hamcrest.CoreMatchers;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.session.Session;
import org.springframework.test.web.servlet.MvcResult;

import javax.servlet.http.Cookie;
import java.net.HttpCookie;
import java.net.URI;
import java.time.Duration;
import java.time.temporal.ChronoUnit;
import java.util.Map;
import java.util.Optional;

import static com.github.nkonev.aaa.TestConstants.XSRF_TOKEN_VALUE;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

public class UserProfileControllerTest extends AbstractUtTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private EventReceiver receiver;


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
        receiver.clear();
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String initialPassword = userAccount.password();

        final String newLogin = "new_alice";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
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

        for (int i=0; i<10; ++i) {
            if (receiver.size() > 0) {
                break;
            } else {
                Uninterruptibles.sleepUninterruptibly(Duration.of(1, ChronoUnit.SECONDS));
            }
        }
        Assertions.assertEquals(1, receiver.size());
        final UserAccountDTO userAccountEvent = receiver.getLast();
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
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
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

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void fullyAuthenticatedUserCannotChangeHerProfileWithoutUsername() throws Exception {
        UserAccount userAccount = getUserFromBd(TestConstants.USER_ALICE);
        final String newPassword = "new_alice_password";

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(null);
        edit = edit.withPassword(newPassword);

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("validation error"))
                .andExpect(jsonPath("$.message").value("empty login"))
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());
    }


    /**
     * Bob wants steal Alice's account by rewrite login and set her id
     * @throws Exception
     */
    @org.junit.jupiter.api.Test
    @WithUserDetails(TestConstants.USER_BOB)
    public void fullyAuthenticatedUserCannotChangeForeignProfile() throws Exception {
        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_ALICE);
        String foreignUserAccountLogin = foreignUserAccount.username();
        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(foreignUserAccount);

        final String badLogin = "stolen";
        edit = edit.withLogin(badLogin);
        Map<String, Object> userMap = objectMapper.readValue(objectMapper.writeValueAsString(edit), new TypeReference<Map<String, Object>>(){} );
        userMap.put("id", foreignUserAccount.id());

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
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

        final String newLogin = TestConstants.USER_BOB;

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withLogin(newLogin);

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
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

        final String newEmail = TestConstants.USER_BOB+"@example.com";
        final Optional<UserAccount> foreignBobAccountOptional = userAccountRepository.findByEmail(newEmail);
        final UserAccount foreignBobAccount = foreignBobAccountOptional.orElseThrow(()->new RuntimeException("foreign email '"+newEmail+"' must be present"));
        final long foreingId = foreignBobAccount.id();
        final String foreignPassword = foreignBobAccount.password();
        final String foreignEmail = foreignBobAccount.email();

        EditUserDTO edit = UserAccountConverter.convertToEditUserDto(userAccount);
        edit = edit.withEmail(newEmail);

        MvcResult mvcResult = mockMvc.perform(
                post(Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE)
                        .content(objectMapper.writeValueAsString(edit))
                        .contentType(MediaType.APPLICATION_JSON_UTF8)
                        .with(csrf())
        )
                .andExpect(status().isOk()) // we care for emails
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

        UserAccount foreignAccountAfter = getUserFromBd(TestConstants.USER_BOB);
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
        UserAccount bob = getUserFromBd(TestConstants.USER_BOB);
        String bobEmail = bob.email();

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.PUBLIC_API + Constants.Urls.USER+Constants.Urls.LIST+"?userId="+bob.id())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].email").doesNotExist())
                .andExpect(jsonPath("$[0].login").value(TestConstants.USER_BOB))
                .andExpect(content().string(CoreMatchers.not(CoreMatchers.containsString(bobEmail))))
                .andReturn();

    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @org.junit.jupiter.api.Test
    public void testGetManyUsers() throws Exception {
        UserAccount bob = getUserFromBd(TestConstants.USER_BOB);
        UserAccount alice = getUserFromBd(TestConstants.USER_ALICE);

        String bobEmail = bob.email();

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.PUBLIC_API + Constants.Urls.USER+Constants.Urls.LIST+"?userId="+bob.id()+"&userId="+alice.id())
        )
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].login").value(TestConstants.USER_ALICE))
                .andExpect(jsonPath("$[1].login").value(TestConstants.USER_BOB))
                .andReturn();

    }


    @org.junit.jupiter.api.Test
    public void userCanSeeOnlyOwnProfileEmail() throws Exception {
        String session = getSession(TestConstants.USER_ALICE, TestConstants.USER_ALICE_PASSWORD);
        String headerValue = buildCookieHeader(new HttpCookie(TestConstants.HEADER_XSRF_TOKEN, XSRF_TOKEN_VALUE), new HttpCookie(getAuthCookieName(), session));

        UserAccount foreignUserAccount = getUserFromBd(TestConstants.USER_BOB);
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
                get(Constants.Urls.PUBLIC_API + Constants.Urls.USER)
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data[2].canDelete").value(false))
                .andExpect(jsonPath("$.data[2].canChangeRole").value(false))
                .andExpect(jsonPath("$.data[2].canLock").value(false))

                .andReturn();
    }

    @WithUserDetails(TestConstants.USER_ADMIN)
    @org.junit.jupiter.api.Test
    public void adminCanManageSessionsView() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                get(Constants.Urls.PUBLIC_API + Constants.Urls.USER)
        )
                .andDo(result -> {
                    LOGGER.info(result.getResponse().getContentAsString());
                })
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data[2].canDelete").value(true))
                .andExpect(jsonPath("$.data[2].canChangeRole").value(true))
                .andExpect(jsonPath("$.data[2].canLock").value(true))

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
                .andExpect(jsonPath("$.users.length()").value(1))
                .andExpect(jsonPath("$.users.[0].login").value("John Smith"))
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
                .andExpect(jsonPath("$.users.length()").value(1))
                .andExpect(jsonPath("$.users.[0].login").value("John Smith"))
                .andReturn();
        String getStr = getPostRequest.getResponse().getContentAsString();
        LOGGER.info(getStr);

    }

    private long createUserForDelete(String login) {
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                login, null, null, null, null,false, false, true,
                UserRole.ROLE_USER, login+"@example.com", null, null);
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
        getSession("bob", "bobspassword");
        Optional<UserAccount> bob = userAccountRepository.findByUsername("bob");
        Assertions.assertTrue(bob.isPresent());
        Map<String, Session> bobRedisSessions = aaaUserDetailsService.getSessions("bob");
        Assertions.assertEquals(1, bobRedisSessions.size());
    }
}
