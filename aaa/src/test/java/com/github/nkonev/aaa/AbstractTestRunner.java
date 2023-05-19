package com.github.nkonev.aaa;

/**
 * Created by nik on 27.05.17.
 */

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.dto.SuccessfulLoginDTO;
import com.github.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import com.github.nkonev.aaa.util.ContextPathHelper;
import com.github.nkonev.oauth2emu.UserTestService;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.extension.ExtendWith;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.web.ServerProperties;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.data.redis.connection.DefaultStringRedisConnection;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.connection.RedisServerCommands;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import javax.annotation.PostConstruct;
import java.net.HttpCookie;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.Arrays;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.CommonTestConstants.*;
import static com.github.nkonev.aaa.security.SecurityConfig.*;
import static org.springframework.http.HttpHeaders.ACCEPT;
import static org.springframework.http.HttpHeaders.COOKIE;

@ExtendWith(SpringExtension.class)
@SpringBootTest(
        classes = {AaaApplication.class, AbstractTestRunner.UtConfig.class},
        webEnvironment = SpringBootTest.WebEnvironment.DEFINED_PORT,
        properties = {
                "spring.config.location=classpath:/config/application.yml,classpath:/config/oauth2-basic.yml,classpath:/config/oauth2-keycloak.yml"
        }
)
@Import(UserTestService.class)
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_CLASS)
public abstract class AbstractTestRunner {

    @Configuration
    public static class UtConfig {

        @Bean(destroyMethod = "close")
        public DefaultStringRedisConnection defaultStringRedisConnection(RedisConnectionFactory redisConnectionFactory){
            return new DefaultStringRedisConnection(redisConnectionFactory.getConnection());
        }

    }

    @Autowired
    protected UserConfirmationTokenRepository userConfirmationTokenRepository;

    @Autowired(required = false)
    protected TestRestTemplate testRestTemplate;

    @Autowired
    protected RestTemplate restTemplate;

    @Value("${custom.base-url}")
    protected String urlPrefix;

    @Autowired
    private ServerProperties serverProperties;

    @Autowired
    protected AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer;

    public String urlWithContextPath(){
        return ContextPathHelper.urlWithContextPath(abstractConfigurableEmbeddedServletContainer);
    }

    @Value(CommonTestConstants.USER)
    protected String username;

    @Value(CommonTestConstants.PASSWORD)
    protected String password;

    @Value(CommonTestConstants.USER_ID)
    protected String userId;

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

    @Autowired
    private RedisServerCommands redisServerCommands;

    @PostConstruct
    public void dropRedis(){
        redisServerCommands.flushDb();
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

    protected SessionHolder login(String login, String password) throws URISyntaxException {
        ResponseEntity<SuccessfulLoginDTO> loginResponseEntity = rawLogin(login, password);

        Assertions.assertEquals(200, loginResponseEntity.getStatusCodeValue());

        return new SessionHolder(loginResponseEntity.getBody().id(), loginResponseEntity);
    }

    protected ResponseEntity<SuccessfulLoginDTO> rawLogin(String login, String password) throws URISyntaxException {
        ResponseEntity<String> getXsrfTokenResponse = testRestTemplate.getForEntity(urlWithContextPath(), String.class);
        String xsrfCookieHeaderValue = getXsrfCookieHeaderValue(getXsrfTokenResponse);
        String xsrf = getXsrfValue(xsrfCookieHeaderValue);


        MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
        params.add(USERNAME_PARAMETER, login);
        params.add(PASSWORD_PARAMETER, password);

        RequestEntity loginRequest = RequestEntity
                .post(new URI(urlWithContextPath()+API_LOGIN_URL))
                .header(HEADER_XSRF_TOKEN, xsrf)
                .header(COOKIE, xsrfCookieHeaderValue)
                .header(ACCEPT, MediaType.APPLICATION_JSON_UTF8_VALUE)
                .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                .body(params);

        return testRestTemplate.exchange(loginRequest, SuccessfulLoginDTO.class);
    }


    protected String getAuthCookieName() {
        return serverProperties.getServlet().getSession().getCookie().getName();
    }

}
