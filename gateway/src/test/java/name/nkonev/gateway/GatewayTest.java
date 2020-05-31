package name.nkonev.gateway;

import name.nkonev.aaa.UserSessionResponse;
import org.junit.jupiter.api.*;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockserver.integration.ClientAndServer;
import org.mockserver.model.BinaryBody;
import org.mockserver.model.Header;
import org.mockserver.model.Headers;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.AutoConfigureWebTestClient;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.cloud.gateway.route.RouteLocator;
import org.springframework.cloud.gateway.route.builder.RouteLocatorBuilder;
import org.springframework.cloud.loadbalancer.annotation.LoadBalancerClient;
import org.springframework.cloud.loadbalancer.core.ServiceInstanceListSupplier;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.env.Environment;
import org.springframework.http.*;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.test.web.reactive.server.WebTestClient;

import java.io.ByteArrayOutputStream;
import java.util.Collections;

import static name.nkonev.gateway.SecurityConfig.APPLICATION_X_PROTOBUF_CHARSET_UTF_8;
import static name.nkonev.gateway.SecurityConfig.SESSION_COOKIE;
import static name.nkonev.gateway.SecurityConfig.SecurityFilter.X_AUTH_USER_ID;
import static name.nkonev.gateway.SecurityConfig.SecurityFilter.X_AUTH_USERNAME;
import static org.mockserver.integration.ClientAndServer.startClientAndServer;
import static org.mockserver.model.HttpRequest.request;
import static org.mockserver.model.HttpResponse.response;

@ExtendWith(SpringExtension.class)
@SpringBootTest(classes = {GatewayApplication.class, GatewayTest.TestGatewayRoutesConfig.class}, properties = {
        "aaa.base-url=http://localhost:9088/internal"
})
@AutoConfigureWebTestClient
public class GatewayTest {

    @Autowired
    private WebTestClient webTestClient;

    private static final Long USER_ID = 42L;
    private static final String USER_ID_STRING = USER_ID.toString();
    private static final String JLONG = "jlong";
    private static final String TRUE = "true";
    private static final String FAKE_SESSION = "fake-session-id-value";
    private static final String CHAT_RESPONSE = "your chats and echoed headers";

    private static final String X_FROM_UPSTREAM = "X-From-Upstream";

    protected static final int DOWNSTREAM_PORT = 9988;
    protected static final int AAA_PORT = 9088;

    static ClientAndServer chatEmulator;
    static ClientAndServer aaaEmulator;

    @BeforeAll
    public static void beforeAll() {
        aaaEmulator = startClientAndServer(AAA_PORT);
        chatEmulator = startClientAndServer(DOWNSTREAM_PORT);
    }

    @AfterAll
    public static void tearDownClass() {
        aaaEmulator.stop();
        chatEmulator.stop();
    }

    @BeforeEach
    public void beforeEach() {
        aaaEmulator.when(request().withPath("/internal/profile")).respond(httpRequest -> {
            // verify SESSION_COOKIE
            String session = httpRequest.getCookies().getEntries().stream().filter(cookie -> SESSION_COOKIE.equals(cookie.getName().getValue())).map(cookie -> cookie.getValue().toString()).findFirst().get();
            if (!FAKE_SESSION.equals(session)) {
                return response().withStatusCode(401);
            }
            UserSessionResponse userSessionResponse = UserSessionResponse.newBuilder()
                    .setUserName(JLONG)
                    .setExpiresIn(0)
                    .addAllRoles(Collections.singletonList("ROLE_USER"))
                    .setUserId(USER_ID)
                    .build();
            ByteArrayOutputStream bs = new ByteArrayOutputStream();
            userSessionResponse.writeTo(bs);
            BinaryBody bb = new BinaryBody(bs.toByteArray(), org.mockserver.model.MediaType.parse(APPLICATION_X_PROTOBUF_CHARSET_UTF_8));
            return response().withBody(bb).withStatusCode(200);
        });

        // chat emulator which echoes headers back
        chatEmulator.when(request().withPath("/chat")).respond(httpRequest -> {
            Header header0 = new Header(X_AUTH_USERNAME, httpRequest.getHeader(X_AUTH_USERNAME).get(0));
            Header header1 = new Header(X_AUTH_USER_ID, httpRequest.getHeader(X_AUTH_USER_ID).get(0));
            Header header2 = new Header(X_FROM_UPSTREAM, TRUE);
            Headers headers = new Headers(header0, header1, header2);
            return response().withBody(CHAT_RESPONSE).withStatusCode(200).withHeaders(headers);
        });
    }

    @AfterEach
    public void afterEach(){
        aaaEmulator.reset();
        chatEmulator.reset();
    }

    @Test
    public void testInsertingHeaders() {
        // invokes the gateway and asserts that authorization has been done by calling aaa
        webTestClient
                // Create a GET request to test an endpoint
                .get().uri("/api/chat")
                .cookie(SESSION_COOKIE, FAKE_SESSION)
                .accept(MediaType.TEXT_PLAIN)
                .exchange()

                .expectStatus().isOk()
                .expectHeader().valueEquals(X_AUTH_USERNAME, JLONG)
                .expectHeader().valueEquals(X_AUTH_USER_ID, USER_ID_STRING)
                .expectHeader().valueEquals(X_FROM_UPSTREAM, TRUE)
                .expectBody(String.class).isEqualTo(CHAT_RESPONSE);
    }

    public static final String UPSTREAM_SERVICE_ID = "chatservice";

    @Configuration
    @LoadBalancerClient(name = UPSTREAM_SERVICE_ID, configuration = TestGatewayLoadBalancerConfig.class)
    public static class TestGatewayRoutesConfig {

        private final static String uri = "lb://"+ UPSTREAM_SERVICE_ID;

        @Bean
        public RouteLocator routeLocator(RouteLocatorBuilder builder) {
            return builder.routes()
                    .route("chat_route", r -> r.path("/api/**").filters(spec -> spec.stripPrefix(1)).uri(uri))
                    .build();
        }
    }

    public static class TestGatewayLoadBalancerConfig {
        @Bean
        public ServiceInstanceListSupplier staticServiceInstanceListSupplier(Environment env) {
            return ServiceInstanceListSupplier.fixed(env).instance(DOWNSTREAM_PORT, UPSTREAM_SERVICE_ID).build();
        }
    }


}
