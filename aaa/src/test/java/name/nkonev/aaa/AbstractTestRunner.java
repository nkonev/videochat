package name.nkonev.aaa;

import com.fasterxml.jackson.databind.ObjectMapper;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.SuccessfulLoginDTO;
import name.nkonev.aaa.dto.AaaError;
import name.nkonev.aaa.repository.redis.ChangeEmailConfirmationTokenRepository;
import name.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import name.nkonev.aaa.util.ContextPathHelper;
import name.nkonev.aaa.services.UserTestService;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.extension.ExtendWith;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.core.RabbitAdmin;
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
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.net.HttpCookie;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.*;
import java.util.stream.Collectors;

import static name.nkonev.aaa.TestConstants.*;
import static name.nkonev.aaa.config.RabbitMqConfig.*;
import static name.nkonev.aaa.config.RabbitMqTestConfig.QUEUE_PROFILE_TEST;
import static name.nkonev.aaa.security.SecurityConfig.*;
import static org.springframework.http.HttpHeaders.ACCEPT;
import static org.springframework.http.HttpHeaders.COOKIE;

@ExtendWith(SpringExtension.class)
@SpringBootTest(
        classes = {AaaApplication.class, AbstractTestRunner.UtConfig.class},
        webEnvironment = SpringBootTest.WebEnvironment.DEFINED_PORT,
        // also see in run-with-oauth2.sh
        properties = {
                "spring.config.location=classpath:/config/application.yml,classpath:/config/oauth2-basic.yml,classpath:/config/oauth2-keycloak.yml,classpath:/config/demo-migration.yml,classpath:/config/user-test-controller.yml,classpath:/config/login-additional-allowed-characters.yml"
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

    @Autowired
    protected ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    @Autowired
    protected AaaProperties aaaProperties;

    @Autowired(required = false)
    protected TestRestTemplate testRestTemplate;

    @Autowired
    protected RestTemplate restTemplate;

    @Value("${custom.template-engine-url-prefix}")
    protected String templateEngineUrlPrefix;

    @Autowired
    private ServerProperties serverProperties;

    @Autowired
    protected AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer;

    public String urlWithContextPath(){
        return ContextPathHelper.urlWithContextPath(abstractConfigurableEmbeddedServletContainer);
    }

    @Value(TestConstants.USER)
    protected String username;

    @Value(TestConstants.PASSWORD)
    protected String password;

    @Value(TestConstants.USER_ID)
    protected String userId;

    @Autowired
    protected ObjectMapper objectMapper;

    @Autowired
    private RedisServerCommands redisServerCommands;

    @Autowired
    protected RabbitAdmin rabbitAdmin;

    @Autowired
    protected JdbcTemplate jdbcTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(AbstractTestRunner.class);

    protected String buildCookieHeader(HttpCookie... cookies) {
        return String.join("; ", Arrays.stream(cookies).map(httpCookie -> httpCookie.toString()).collect(Collectors.toList()));
    }

    @BeforeEach
    public void before() {
        redisServerCommands.flushDb();
        userConfirmationTokenRepository.deleteAll();
        rabbitAdmin.purgeQueue(QUEUE_USER_CONFIRMATION_EMAILS_NAME, true);
        rabbitAdmin.purgeQueue(QUEUE_PASSWORD_RESET_EMAILS_NAME, true);
        rabbitAdmin.purgeQueue(QUEUE_CHANGE_EMAIL_CONFIRMATION_NAME, true);
        rabbitAdmin.purgeQueue(QUEUE_ARBITRARY_EMAILS_NAME, true);
        rabbitAdmin.purgeQueue(QUEUE_PROFILE_TEST, true);
        jdbcTemplate.execute("delete from user_account where ldap_id is not null");
        jdbcTemplate.execute("delete from user_account where keycloak_id is not null");
    }

    public static class SessionHolder {
        public final long userId;
        final List<String> sessionCookies;
        public String newXsrf;

        SessionHolder(long userId, List<String> sessionCookies, String newXsrf) {
            this.userId = userId;
            this.sessionCookies = sessionCookies;
            this.newXsrf = newXsrf;
        }

        public String[] getCookiesArray(){
            return sessionCookies.toArray(new String[sessionCookies.size()]);
        }

    }

    private static String normalizeCookie(String stringCookie) {
        var parsed = HttpCookie.parse(stringCookie);
        if (parsed.size() != 1) {
            return null;
        }
        var aCookie = parsed.get(0);
        return aCookie.getName() + "=" + aCookie.getValue();
    }

    public static List<String> getSessionCookies(ResponseEntity<?> loginResponseEntity) {
        return getSetCookieHeaders(loginResponseEntity).stream().dropWhile(s -> s.contains(COOKIE_XSRF+"=;")).map(AbstractTestRunner::normalizeCookie).filter(Objects::nonNull).collect(Collectors.toList());
    }

    public static List<String> getSessionIdCookie(List<String> cookies) {
        return cookies.stream().filter(s -> s.matches(SESSION_COOKIE_NAME+"=\\w+.*")).toList();
    }

    public static String getXsrfValue(String xsrfCookieHeaderValue) {
        return HttpCookie.parse(xsrfCookieHeaderValue).stream().findFirst().orElseThrow(()-> new RuntimeException("cannot get cookie value")).getValue();
    }

    public static String getXsrfCookieHeaderValue(ResponseEntity<String> getXsrfTokenResponse) {
        return getSetCookieHeaders(getXsrfTokenResponse)
                .stream().filter(s -> s.matches(COOKIE_XSRF+"=\\w+.*"))
                .map(AbstractTestRunner::normalizeCookie)
                .filter(Objects::nonNull)
                .findFirst().orElseThrow(()-> new RuntimeException("cookie " + COOKIE_XSRF + " not found"));
    }

    public static List<String> getSetCookieHeaders(ResponseEntity<?> getXsrfTokenResponse) {
        return Optional.ofNullable(getXsrfTokenResponse.getHeaders().get(HEADER_SET_COOKIE)).orElseThrow(()->new RuntimeException("missed header "+ HEADER_SET_COOKIE));
    }


    public static class XsrfCookiesHolder {
        final public List<String> sessionCookies;
        final public String newXsrf;

        final public String xsrfCookieHeaderValue;

        public XsrfCookiesHolder(List<String> sessionCookies, String newXsrf, String xsrfCookieHeaderValue) {
            this.sessionCookies = sessionCookies;
            this.newXsrf = newXsrf;
            this.xsrfCookieHeaderValue = xsrfCookieHeaderValue;
        }

    }

    protected XsrfCookiesHolder getXsrf() {
        var url = urlWithContextPath();
        url += "/login.html";
        var bldr = RequestEntity.get(url);
        var reqEntity = bldr.build();

        ResponseEntity<String> getXsrfTokenResponse = testRestTemplate.exchange(reqEntity, String.class);
        String xsrfCookieHeaderValue = getXsrfCookieHeaderValue(getXsrfTokenResponse);
        String xsrf = getXsrfValue(xsrfCookieHeaderValue);
        List<String> sessionCookies = getSessionCookies(getXsrfTokenResponse);
        return new XsrfCookiesHolder(sessionCookies, xsrf, xsrfCookieHeaderValue);
    }

    protected SessionHolder login(String login, String password) throws URISyntaxException {
        var rawLoginResponse = rawLogin(login, password);

        Assertions.assertEquals(200, rawLoginResponse.dto.getStatusCodeValue());

        var sessionIdCookies = getSessionIdCookie(getSessionCookies(rawLoginResponse.dto));

        var respondableSessionCookies = new ArrayList<String>();
        respondableSessionCookies.addAll(sessionIdCookies);
        respondableSessionCookies.add(COOKIE_XSRF + "=" + rawLoginResponse.xsrfHolder.newXsrf);

        return new SessionHolder(rawLoginResponse.dto.getBody().id(), respondableSessionCookies, rawLoginResponse.xsrfHolder.newXsrf);
    }

    public record RawLoginResponse(
        ResponseEntity<SuccessfulLoginDTO> dto,
        XsrfCookiesHolder xsrfHolder
    ) {}

    public record RawLoginErrorResponse(
        ResponseEntity<AaaError> dto,
        XsrfCookiesHolder xsrfHolder
    ) {}

    protected RawLoginResponse rawLogin(String login, String password) {
        var xsrfHolder = getXsrf();
        String xsrfCookieHeaderValue = xsrfHolder.xsrfCookieHeaderValue;
        String xsrf = xsrfHolder.newXsrf;

        MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
        params.add(USERNAME_PARAMETER, login);
        params.add(PASSWORD_PARAMETER, password);

        RequestEntity loginRequest = RequestEntity
            .post(URI.create(urlWithContextPath()+API_LOGIN_URL))
            .header(HEADER_XSRF_TOKEN, xsrf)
            .header(COOKIE, xsrfCookieHeaderValue)
            .header(ACCEPT, MediaType.APPLICATION_JSON_UTF8_VALUE)
            .contentType(MediaType.APPLICATION_FORM_URLENCODED)
            .body(params);

        ResponseEntity<SuccessfulLoginDTO> loginResponseEntity = testRestTemplate.exchange(loginRequest, SuccessfulLoginDTO.class);
        return new RawLoginResponse(loginResponseEntity, xsrfHolder);
    }

    protected RawLoginErrorResponse rawLoginDecodeError(String login, String password) {
        var xsrfHolder = getXsrf();
        String xsrfCookieHeaderValue = xsrfHolder.xsrfCookieHeaderValue;
        String xsrf = xsrfHolder.newXsrf;

        MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
        params.add(USERNAME_PARAMETER, login);
        params.add(PASSWORD_PARAMETER, password);

        RequestEntity loginRequest = RequestEntity
            .post(URI.create(urlWithContextPath()+API_LOGIN_URL))
            .header(HEADER_XSRF_TOKEN, xsrf)
            .header(COOKIE, xsrfCookieHeaderValue)
            .header(ACCEPT, MediaType.APPLICATION_JSON_UTF8_VALUE)
            .contentType(MediaType.APPLICATION_FORM_URLENCODED)
            .body(params);

        ResponseEntity<AaaError> loginResponseEntity = testRestTemplate.exchange(loginRequest, AaaError.class);
        return new RawLoginErrorResponse(loginResponseEntity, xsrfHolder);
    }

    protected String getAuthCookieName() {
        return serverProperties.getServlet().getSession().getCookie().getName();
    }

}
