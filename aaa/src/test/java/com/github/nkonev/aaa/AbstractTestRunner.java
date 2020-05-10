package com.github.nkonev.aaa;

/**
 * Created by nik on 27.05.17.
 */

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import com.github.nkonev.aaa.security.SecurityConfig;
import com.github.nkonev.aaa.util.ContextPathHelper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.extension.ExtendWith;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.DefaultStringRedisConnection;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.connection.RedisServerCommands;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.web.client.RestTemplate;

import javax.annotation.PostConstruct;
import java.net.HttpCookie;
import java.util.Arrays;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.CommonTestConstants.COOKIE_XSRF;
import static com.github.nkonev.aaa.CommonTestConstants.HEADER_SET_COOKIE;
import static com.github.nkonev.aaa.security.SecurityConfig.PASSWORD_PARAMETER;
import static com.github.nkonev.aaa.security.SecurityConfig.USERNAME_PARAMETER;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@ExtendWith(SpringExtension.class)
@SpringBootTest(
        classes = {AaaApplication.class, AbstractTestRunner.UtConfig.class},
        webEnvironment = SpringBootTest.WebEnvironment.DEFINED_PORT,
        properties = {"spring.test.context.cache.maxSize=1"}
)
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_CLASS)
public abstract class AbstractTestRunner {

    @Configuration
    public static class UtConfig {

        @Autowired
        private RedisServerCommands redisServerCommands;

        @Bean(destroyMethod = "close")
        public DefaultStringRedisConnection defaultStringRedisConnection(RedisConnectionFactory redisConnectionFactory){
            return new DefaultStringRedisConnection(redisConnectionFactory.getConnection());
        }

        @PostConstruct
        public void dropRedis(){
            redisServerCommands.flushDb();
        }
    }

    @Autowired
    protected UserConfirmationTokenRepository userConfirmationTokenRepository;

    @Autowired
    protected TestRestTemplate testRestTemplate;

    @Autowired
    protected RestTemplate restTemplate;

    @Value("${local.management.port}")
    protected int mgmtPort;

    @Value("${custom.base-url}")
    protected String urlPrefix;

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

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractTestRunner.class);

    protected String buildCookieHeader(HttpCookie... cookies) {
        return String.join("; ", Arrays.stream(cookies).map(httpCookie -> httpCookie.toString()).collect(Collectors.toList()));
    }

    @BeforeEach
    public void before() {
        userConfirmationTokenRepository.deleteAll();
    }

    public static class SessionHolder {
        public final long userId;
        final List<String> sessionCookies;
        public String newXsrf;

        public SessionHolder(long userId, List<String> sessionCookies, String newXsrf) {
            this.userId = userId;
            this.sessionCookies = sessionCookies;
            this.newXsrf = newXsrf;
        }

        public SessionHolder(long userId, ResponseEntity responseEntity) {
            this.userId = userId;
            this.sessionCookies = getSessionCookies(responseEntity);
            this.newXsrf = getXsrfValue(getXsrfCookieHeaderValue(responseEntity));
        }

        public String[] getCookiesArray(){
            return sessionCookies.toArray(new String[sessionCookies.size()]);
        }

        public void updateXsrf(ResponseEntity responseEntity){
            this.newXsrf = getXsrfValue(getXsrfCookieHeaderValue(responseEntity));
        }
    }

    public static List<String> getSessionCookies(ResponseEntity<String> loginResponseEntity) {
        return getSetCookieHeaders(loginResponseEntity).stream().dropWhile(s -> s.contains(COOKIE_XSRF+"=;")).collect(Collectors.toList());
    }

    public static String getXsrfValue(String xsrfCookieHeaderValue) {
        return HttpCookie.parse(xsrfCookieHeaderValue).stream().findFirst().orElseThrow(()-> new RuntimeException("cannot get cookie value")).getValue();
    }

    public static String getXsrfCookieHeaderValue(ResponseEntity<String> getXsrfTokenResponse) {
        return getSetCookieHeaders(getXsrfTokenResponse)
                .stream().filter(s -> s.matches(COOKIE_XSRF+"=\\w+.*")).findFirst().orElseThrow(()-> new RuntimeException("cookie " + COOKIE_XSRF + " not found"));
    }

    public static List<String> getSetCookieHeaders(ResponseEntity<String> getXsrfTokenResponse) {
        return Optional.ofNullable(getXsrfTokenResponse.getHeaders().get(HEADER_SET_COOKIE)).orElseThrow(()->new RuntimeException("missed header "+ HEADER_SET_COOKIE));
    }


    public static class XsrfCookiesHolder {
        final List<String> sessionCookies;
        final public String newXsrf;


        public XsrfCookiesHolder(List<String> sessionCookies, String newXsrf) {
            this.sessionCookies = sessionCookies;
            this.newXsrf = newXsrf;
        }

        public String[] getCookiesArray(){
            return sessionCookies.toArray(new String[sessionCookies.size()]);
        }
    }

    public XsrfCookiesHolder getXsrf() {
        ResponseEntity<String> getXsrfTokenResponse = testRestTemplate.getForEntity(urlWithContextPath(), String.class);
        String xsrfCookieHeaderValue = getXsrfCookieHeaderValue(getXsrfTokenResponse);
        String xsrf = getXsrfValue(xsrfCookieHeaderValue);
        List<String> sessionCookies = getSessionCookies(getXsrfTokenResponse);
        return new XsrfCookiesHolder(sessionCookies, xsrf);
    }
}
