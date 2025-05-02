package name.nkonev.aaa.config;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import name.nkonev.aaa.AbstractMockMvcTestRunner;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.TestConstants;
import name.nkonev.aaa.dto.AdditionalDataDTO;
import name.nkonev.aaa.dto.OAuth2IdentifiersDTO;
import name.nkonev.aaa.dto.UserAccountDTO;
import name.nkonev.aaa.dto.UserRole;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.test.context.support.WithUserDetails;
import org.springframework.test.web.servlet.MvcResult;

import java.time.LocalDateTime;
import java.time.Month;
import java.util.Set;

import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class JacksonTest extends AbstractMockMvcTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(JacksonTest.class);

    @Autowired
    private ObjectMapper objectMapper;

    @Test
    public void testDateWoMillis() throws JsonProcessingException {
        var s = objectMapper.writeValueAsString(new UserAccountDTO(
                1L,
                "user",
                null,
                null,
                null,
                LocalDateTime.of(2024, Month.APRIL, 29, 19, 55, 56, 0),
                new OAuth2IdentifiersDTO(),
                null,
                false,
                new AdditionalDataDTO(true , false, false, true, Set.of(UserRole.ROLE_USER))
        ));
        LOGGER.info(s);
        Assertions.assertTrue(s.contains("2024-04-29T19:55:56.000Z"));
    }

    @Test
    public void testDateWAllMillis() throws JsonProcessingException {
        var s = objectMapper.writeValueAsString(new UserAccountDTO(
                1L,
                "user",
                null,
                null,
                null,
                LocalDateTime.of(2024, Month.APRIL, 29, 19, 55, 56, 123000000),
                new OAuth2IdentifiersDTO(),
                null,
                false,
                new AdditionalDataDTO(true , false, false, true, Set.of(UserRole.ROLE_USER))
        ));
        LOGGER.info(s);
        Assertions.assertTrue(s.contains("2024-04-29T19:55:56.123Z"));
    }

    @Test
    public void testDateWSomeMillis() throws JsonProcessingException {
        var s = objectMapper.writeValueAsString(new UserAccountDTO(
                1L,
                "user",
                null,
                null,
                null,
                LocalDateTime.of(2024, Month.APRIL, 29, 19, 55, 56, 120000000),
                new OAuth2IdentifiersDTO(),
                null,
                false,
                new AdditionalDataDTO(true , false, false, true, Set.of(UserRole.ROLE_USER))
        ));
        LOGGER.info(s);
        Assertions.assertTrue(s.contains("2024-04-29T19:55:56.120Z"));
    }

    @WithUserDetails(TestConstants.USER_ALICE)
    @Test
    public void freshWithExternallyProvidedTimestampWithVariableFractionSeconds() throws Exception {

        MvcResult mvcResult = mockMvc.perform(
                        post(Constants.Urls.EXTERNAL_API + Constants.Urls.USER + Constants.Urls.FRESH)
                                .param("size", "40")
                                .content("""
                                        [
                                          {
                                            "id": 1007,
                                            "login": "forgot-password-user",
                                            "avatar": "/api/storage/public/user/avatar/1007_AVATAR_200x200.jpg?time=1746128796",
                                            "avatarBig": "/api/storage/public/user/avatar/1007_AVATAR_640x640.jpg?time=1746128796",
                                            "shortInfo": "biba",
                                            "lastSeenDateTime": "2025-05-01T19:51:43.79Z",
                                            "oauth2Identifiers": {
                                              "facebookId": null,
                                              "vkontakteId": null,
                                              "googleId": null,
                                              "keycloakId": null
                                            },
                                            "additionalData": {
                                              "enabled": true,
                                              "expired": false,
                                              "locked": true,
                                              "confirmed": true,
                                              "roles": [
                                                "ROLE_USER"
                                              ]
                                            },
                                            "canLock": true,
                                            "canEnable": true,
                                            "canDelete": true,
                                            "canChangeRole": true,
                                            "canConfirm": true,
                                            "loginColor": null,
                                            "canRemoveSessions": true,
                                            "ldap": false,
                                            "canSetPassword": true,
                                            "online": false,
                                            "isInVideo": false
                                          }
                                        ]
                                        """)
                                .contentType(MediaType.APPLICATION_JSON_UTF8)
                                .with(csrf())
                )
                .andExpect(status().isOk())
                .andReturn();

        LOGGER.info(mvcResult.getResponse().getContentAsString());

    }

}
