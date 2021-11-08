package com.github.nkonev.aaa;

/**
 * Created by nik on 27.05.17.
 */

import com.github.nkonev.aaa.security.SecurityConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.MockMvcPrint;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.transaction.annotation.Transactional;

import static com.github.nkonev.aaa.security.SecurityConfig.PASSWORD_PARAMETER;
import static com.github.nkonev.aaa.security.SecurityConfig.USERNAME_PARAMETER;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@AutoConfigureMockMvc(printOnlyOnFailure = false, print = MockMvcPrint.LOG_DEBUG)
@Transactional
public abstract class AbstractUtTestRunner extends AbstractTestRunner {


    @Autowired
    protected MockMvc mockMvc;

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractUtTestRunner.class);

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

        return mvcResult.getResponse().getCookie(getAuthCookieName()).getValue();
    }

}
