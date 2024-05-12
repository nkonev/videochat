package name.nkonev.aaa.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import name.nkonev.aaa.AbstractUtTestRunner;
import name.nkonev.aaa.TestConstants;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.security.SecurityConfig;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.*;
import org.springframework.test.web.servlet.MvcResult;

import java.util.Map;
import static name.nkonev.aaa.security.SecurityConfig.PASSWORD_PARAMETER;
import static name.nkonev.aaa.security.SecurityConfig.USERNAME_PARAMETER;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class AaaErrorControllerTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaErrorControllerTest.class);

    @org.junit.jupiter.api.Test
    public void testAuth() throws Exception {
        // auth
        MvcResult loginResult = mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(USERNAME_PARAMETER, username)
                        .param(PASSWORD_PARAMETER, password)
                        .with(csrf())
        )
                .andExpect(status().isOk())
                .andReturn();

        LOGGER.info(loginResult.getResponse().getContentAsString());
    }

    /**
     * We use restTemplate because Spring Security has own exception handling mechanism (not Spring MVC Exception Handler)
     * which eventually handled on Error Page
     * @throws Exception
     */
    @Test
    public void testNotAuthorized() throws Exception {
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.PUBLIC_API + Constants.Urls.PROFILE, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(401, responseEntity.getStatusCodeValue());

        LOGGER.info(str);

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>(){});
        // check that Exception Handler hides Spring Security exceptions
        Assertions.assertFalse(resp.containsKey("exception"));
        Assertions.assertFalse(resp.containsKey("trace"));
        Assertions.assertFalse(resp.containsValue("org.springframework.security.access.AccessDeniedException"));

        Assertions.assertTrue(resp.containsKey("message"));
        Assertions.assertNotNull(resp.get("message"));
    }

    @org.junit.jupiter.api.Test
    public void testNotFoundJs() throws Exception {
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+"/not-exists", String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(404, responseEntity.getStatusCodeValue());

        LOGGER.info(str);

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>(){});

        Assertions.assertTrue(responseEntity.getHeaders().getContentType().toString().contains(MediaType.APPLICATION_JSON_UTF8_VALUE));
        Assertions.assertEquals("data not found", resp.get("error"));
        Assertions.assertEquals(404, resp.get("status"));
    }

    @Test
    public void testSqlExceptionIsHidden() throws Exception {
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.PUBLIC_API + TestConstants.SQL_URL, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(500, responseEntity.getStatusCodeValue());

        LOGGER.info(str);

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>(){});
        Assertions.assertFalse(resp.containsKey("exception"));
        Assertions.assertFalse(resp.containsKey("trace"));
        Assertions.assertFalse(resp.containsValue(TestConstants.SQL_QUERY));

        Assertions.assertEquals("internal error", resp.get("message"));
        Assertions.assertEquals("Internal Server Error", resp.get("error"));
    }

    @Test
    public void testUserDetailsWithPasswordIsNotSerialized() throws Exception {
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.PUBLIC_API + TestConstants.USER_DETAILS, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(500, responseEntity.getStatusCodeValue());

        LOGGER.info(str);

        Map<String, Object> resp = objectMapper.readValue(str, new TypeReference<Map<String, Object>>(){});
        Assertions.assertFalse(resp.containsKey("exception"));
        Assertions.assertFalse(resp.containsKey("trace"));

        Assertions.assertEquals("internal error", resp.get("message"));
        Assertions.assertEquals("Internal Server Error", resp.get("error"));
    }
}
