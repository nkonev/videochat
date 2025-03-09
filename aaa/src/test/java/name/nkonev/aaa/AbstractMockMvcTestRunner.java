package name.nkonev.aaa;

import name.nkonev.aaa.security.SecurityConfig;
import com.icegreen.greenmail.configuration.GreenMailConfiguration;
import com.icegreen.greenmail.junit5.GreenMailExtension;
import com.icegreen.greenmail.util.ServerSetup;
import org.junit.jupiter.api.extension.RegisterExtension;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.MockMvcPrint;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.transaction.annotation.Transactional;

import static name.nkonev.aaa.security.SecurityConfig.PASSWORD_PARAMETER;
import static name.nkonev.aaa.security.SecurityConfig.USERNAME_PARAMETER;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@AutoConfigureMockMvc(printOnlyOnFailure = false, print = MockMvcPrint.LOG_DEBUG)
@Transactional
public abstract class AbstractMockMvcTestRunner extends AbstractTestRunner {


    @Autowired
    protected MockMvc mockMvc;

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractMockMvcTestRunner.class);

    /**
     * This method changes in runtime with ReflectionUtils Spring Security Csrf Filter .with(csrf()) so it ignores any CSRF token
     * @param login
     * @param password
     * @return
     * @throws Exception
     */
    protected String getMockMvcSession(String login, String password) throws Exception {
        MvcResult mvcResult = mockMvc.perform(
                post(SecurityConfig.API_LOGIN_URL)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .param(USERNAME_PARAMETER, login)
                        .param(PASSWORD_PARAMETER, password)
                        .with(csrf())
        ).andDo(mvcResult1 -> {
            LOGGER.info(mvcResult1.getResponse().getContentAsString());
        })
                .andExpect(status().isOk())
                .andReturn();

        return mvcResult.getResponse().getCookie(getAuthCookieName()).getValue();
    }

    private static final int portOffset = 30000; // see also spring.mail.port in src/test/resources/config/application.yml
    private static final ServerSetup SMTP = new ServerSetup(portOffset + 25, null, ServerSetup.PROTOCOL_SMTP);
    private static final ServerSetup IMAP = new ServerSetup(portOffset + 143, null, ServerSetup.PROTOCOL_IMAP);
    private static final ServerSetup[] SMTP_IMAP = new ServerSetup[]{SMTP, IMAP};

    @RegisterExtension
    protected static GreenMailExtension greenMail = new GreenMailExtension(SMTP_IMAP).withConfiguration(GreenMailConfiguration.aConfig().withDisabledAuthentication());

}
