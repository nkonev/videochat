package com.github.nkonev.blog;

/**
 * Created by nik on 27.05.17.
 */

import static com.github.nkonev.blog.security.SecurityConfig.PASSWORD_PARAMETER;
import static com.github.nkonev.blog.security.SecurityConfig.USERNAME_PARAMETER;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.config.UtConfig;
import com.github.nkonev.blog.repository.redis.UserConfirmationTokenRepository;
import com.github.nkonev.blog.security.SecurityConfig;
import com.github.nkonev.blog.util.ContextPathHelper;
import java.net.HttpCookie;
import java.util.Arrays;
import java.util.stream.Collectors;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.extension.ExtendWith;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.MockMvcPrint;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;
import org.springframework.http.MediaType;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.client.RestTemplate;

@ExtendWith(SpringExtension.class)
@SpringBootTest(
        classes = {AaaApplication.class, UtConfig.class},
        webEnvironment = SpringBootTest.WebEnvironment.DEFINED_PORT
)
@AutoConfigureMockMvc(printOnlyOnFailure = false, print = MockMvcPrint.LOG_DEBUG)
@Transactional
public abstract class AbstractUtTestRunner {

    @Autowired
    protected MockMvc mockMvc;

    @Autowired
    protected UserConfirmationTokenRepository userConfirmationTokenRepository;

    @Autowired
    protected TestRestTemplate testRestTemplate;

    @Autowired
    protected RestTemplate restTemplate;

    @Value("${local.management.port}")
    protected int mgmtPort;

    @Autowired
    protected AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer;

    public String urlWithContextPath(){
        return ContextPathHelper.urlWithContextPath(abstractConfigurableEmbeddedServletContainer);
    }

    @Value(CommonTestConstants.USER)
    protected String username;

    @Value(CommonTestConstants.PASSWORD)
    protected String password;

    @Autowired
    protected ObjectMapper objectMapper;

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractUtTestRunner.class);

    protected String buildCookieHeader(HttpCookie... cookies) {
        return String.join("; ", Arrays.stream(cookies).map(httpCookie -> httpCookie.toString()).collect(Collectors.toList()));
    }

    /**
     * This method changes in runtime with ReflectionUtils Spring Security Csrf Filter .with(csrf()) so it ignores any CSRF token
     * @param xsrf
     * @param username
     * @param password
     * @return
     * @throws Exception
     */
    protected String getSession(String xsrf, String username, String password) throws Exception {
        MvcResult mvcResult = mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(USERNAME_PARAMETER, username)
                        .param(PASSWORD_PARAMETER, password)
                        .with(csrf())
        ).andDo(mvcResult1 -> {
            LOGGER.info(mvcResult1.getResponse().getContentAsString());
        })
                .andExpect(status().isOk())
                .andReturn();

        return mvcResult.getResponse().getCookie(CommonTestConstants.COOKIE_SESSION).getValue();
    }

    @BeforeEach
    public void before() {
        userConfirmationTokenRepository.deleteAll();
    }
}
