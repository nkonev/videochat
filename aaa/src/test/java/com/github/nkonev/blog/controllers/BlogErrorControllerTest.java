package com.github.nkonev.blog.controllers;

import com.fasterxml.jackson.core.type.TypeReference;
import com.github.nkonev.blog.AbstractUtTestRunner;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.TestConstants;
import com.github.nkonev.blog.security.SecurityConfig;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.*;
import org.springframework.test.web.servlet.MvcResult;
import java.net.URI;
import java.util.ArrayList;
import java.util.Map;

import static com.github.nkonev.blog.security.SecurityConfig.*;
import static org.springframework.http.HttpMethod.GET;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

public class BlogErrorControllerTest extends AbstractUtTestRunner {

    private static final Logger LOGGER = LoggerFactory.getLogger(BlogErrorControllerTest.class);

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
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.API+ Constants.Urls.PROFILE, String.class);
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
        Assertions.assertEquals("Not Found", resp.get("error"));
        Assertions.assertEquals(404, resp.get("status"));
    }

    @Test
    public void test404FallbackAccept() throws Exception {
        RequestEntity<Void> requestEntity = RequestEntity.<Void>get(new URI(urlWithContextPath()+"/not-exists")).accept(MediaType.TEXT_HTML).build();

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(requestEntity, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(200, responseEntity.getStatusCodeValue()); // we respond 200 for 404 fallback

        LOGGER.info(str);
        LOGGER.info("HTML 404 fallback Content-Type: {}", responseEntity.getHeaders().getContentType());
        Assertions.assertTrue(responseEntity.getHeaders().getContentType().toString().contains(MediaType.TEXT_HTML_VALUE));
        Assertions.assertTrue(str.contains("<!doctype html>"));
    }

    @org.junit.jupiter.api.Test
    public void test404FallbackNoAccept() throws Exception {
        HttpHeaders headers = new HttpHeaders();
        headers.setAccept(new ArrayList<>()); // explicitly set zero Accept values
        HttpEntity<String> entity = new HttpEntity<>(headers);

        ResponseEntity<String> responseEntity = testRestTemplate.exchange(new URI(urlWithContextPath()+"/not-exists"), GET, entity, String.class);
        String str = responseEntity.getBody();
        Assertions.assertEquals(200, responseEntity.getStatusCodeValue()); // we respond 200 for 404 fallback

        LOGGER.info(str);
        LOGGER.info("HTML 404 fallback Content-Type: {}", responseEntity.getHeaders().getContentType());
        Assertions.assertTrue(responseEntity.getHeaders().getContentType().toString().contains(MediaType.TEXT_HTML_VALUE));
        Assertions.assertTrue(str.contains("<!doctype html>"));
    }


    @Test
    public void testSqlExceptionIsHidden() throws Exception {
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.API+ TestConstants.SQL_URL, String.class);
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
        ResponseEntity<String> responseEntity = testRestTemplate.getForEntity(urlWithContextPath()+ Constants.Urls.API+TestConstants.USER_DETAILS, String.class);
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
