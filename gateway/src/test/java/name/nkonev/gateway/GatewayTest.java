package name.nkonev.gateway;

import org.junit.jupiter.api.*;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockserver.integration.ClientAndServer;
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
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.test.web.reactive.server.WebTestClient;
import static name.nkonev.gateway.SecurityConfig.SecurityFilter.X_AUTH_SUBJECT;
import static name.nkonev.gateway.SecurityConfig.SecurityFilter.X_AUTH_USERNAME;
import static org.mockserver.integration.ClientAndServer.startClientAndServer;
import static org.mockserver.model.HttpRequest.request;
import static org.mockserver.model.HttpResponse.response;

// https://howtodoinjava.com/spring-webflux/webfluxtest-with-webtestclient/
@ExtendWith(SpringExtension.class)
@SpringBootTest(classes = {GatewayApplication.class, GatewayTest.TestGatewayRoutesConfig.class})
@AutoConfigureWebTestClient
public class GatewayTest {

    @Autowired
    private WebTestClient webTestClient;

    private static final String JLONG = "jlong";
    private static final String TRUE = "true";

    private static final String X_FROM_DOWNSTREAM = "X-From-Downstream";

    protected static final int DOWNSTREAM_PORT = 9988;

    static ClientAndServer mockServer;

    @BeforeAll
    public static void beforeAll() {
        mockServer = startClientAndServer(DOWNSTREAM_PORT);
    }

    @AfterAll
    public static void tearDownClass() {
        mockServer.stop();
    }

    @BeforeEach
    public void beforeEach() {
        mockServer.when(request().withPath("/api/profit")).respond(httpRequest -> {
            Header header0 = new Header(X_AUTH_USERNAME, httpRequest.getHeader(X_AUTH_USERNAME).get(0));
            Header header1 = new Header(X_AUTH_SUBJECT, httpRequest.getHeader(X_AUTH_SUBJECT).get(0));
            Header header2 = new Header(X_FROM_DOWNSTREAM, TRUE);
            Headers headers = new Headers(header0, header1, header2);
            return response().withBody("done").withStatusCode(200).withHeaders(headers);
        });
    }

    @AfterEach
    public void resetFacebookEmulator(){
        mockServer.reset();
    }

    @WithMockUser(username = JLONG)
    @Test
    public void testInsertingHeaders() {
        webTestClient
                // Create a GET request to test an endpoint
                .get().uri("/api/profit")
                .accept(MediaType.TEXT_PLAIN)
                .exchange()
                // and use the dedicated DSL to test assertions against the response
                .expectStatus().isOk()
                .expectHeader().valueEquals(X_AUTH_USERNAME, JLONG)
                .expectHeader().valueEquals(X_AUTH_SUBJECT, JLONG)
                .expectHeader().valueEquals(X_FROM_DOWNSTREAM, TRUE)
                .expectBody(String.class).isEqualTo("done");
    }

    public static final String DOWNSTREAM_SERVICE_ID = "testservice";

    @Configuration
    @LoadBalancerClient(name = DOWNSTREAM_SERVICE_ID, configuration = TestGatewayLoadBalancerConfig.class)
    public static class TestGatewayRoutesConfig {

        private final static String uri = "lb://"+ DOWNSTREAM_SERVICE_ID;

        @Bean
        public RouteLocator routeLocator(RouteLocatorBuilder builder) {
            return builder.routes()
                    .route("profit_route", r -> r.path("/api/**").uri(uri))
                    .build();
        }
    }

    public static class TestGatewayLoadBalancerConfig {
        @Bean
        public ServiceInstanceListSupplier staticServiceInstanceListSupplier(Environment env) {
            return ServiceInstanceListSupplier.fixed(env).instance(DOWNSTREAM_PORT, DOWNSTREAM_SERVICE_ID).build();
        }
    }


}
