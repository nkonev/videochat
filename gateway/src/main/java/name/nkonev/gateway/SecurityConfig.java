package name.nkonev.gateway;

import name.nkonev.aaa.UserSessionResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.cloud.gateway.route.Route;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.http.HttpCookie;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.util.MultiValueMap;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.server.ServerWebExchange;
import reactor.core.publisher.Mono;

import java.util.Optional;

import static org.springframework.cloud.gateway.support.ServerWebExchangeUtils.GATEWAY_ROUTE_ATTR;
import static org.springframework.cloud.gateway.support.ServerWebExchangeUtils.setAlreadyRouted;

@Configuration
public class SecurityConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityConfig.class);
    public static final String SESSION_COOKIE = "SESSION";

    @Bean
    public WebClient webClient() {
        return WebClient
                .builder()
                .baseUrl("http://localhost:8060/api")
                .defaultHeader(HttpHeaders.ACCEPT, "application/x-protobuf;charset=UTF-8")
                .build();
    }

    @Bean
    public SecurityFilter headerInserter(WebClient webClient) {
        return new SecurityFilter(webClient);
    }

    // inserted before NettyRoutingFilter which containing http client
    public static class SecurityFilter implements GlobalFilter, Ordered {

        private final WebClient client;

        public static final String X_AUTH_USERNAME = "X-Auth-Username";
        public static final String X_AUTH_SUBJECT = "X-Auth-UserId";
        public static final String X_AUTH_EXPIRESIN = "X-Auth-Expiresin";

        public SecurityFilter(WebClient client) {
            this.client = client;
        }

        @Override
        public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
            Optional<String> maybeSessionCookie = getSessionCookie(exchange.getRequest().getCookies());

            if (isSecuredPath(exchange) && !isAaa(exchange)) {
                String session = maybeSessionCookie.orElse(""); // let aaa respond error
                return client
                        .get()
                        .uri("/profile")
                        .cookie(SESSION_COOKIE, session)
                        .exchange()
                        .flatMap(response -> {
                            HttpStatus statusCode = response.statusCode();
                            if (statusCode.value() == 401) {
                                return response.releaseBody().then(Mono.error(new SetStatusException("AAA Unauthorized", statusCode.value())));
                            }

                            return response
                                    .bodyToMono(UserSessionResponse.class)
                                    .switchIfEmpty(Mono.error(new RuntimeException("Empty body from AAA")))
                                    .flatMap(sessionResponse -> {
                                        String username = sessionResponse.getUserName();
                                        long userid = sessionResponse.getUserId();
                                        long expiresIn = sessionResponse.getExpiresIn();

                                        ServerWebExchange modifiedExchange = exchange.mutate().request(builder -> {
                                            builder.header(X_AUTH_USERNAME, username);
                                            builder.header(X_AUTH_SUBJECT, "" + userid);
                                            builder.header(X_AUTH_EXPIRESIN, "" + expiresIn);
                                        }).build();
                                        LOGGER.info("Into {} '{}' inserting {}='{}', {}='{}', {}='{}'",
                                                modifiedExchange.getRequest().getMethod(),
                                                modifiedExchange.getRequest().getURI(),
                                                X_AUTH_USERNAME, username,
                                                X_AUTH_SUBJECT, userid,
                                                X_AUTH_EXPIRESIN, expiresIn
                                        );
                                        return chain.filter(modifiedExchange);
                                    });
                        })
                        .onErrorResume(throwable -> {
                            setAlreadyRouted(exchange);
                            exchange.getResponse().setRawStatusCode(500);
                            if (throwable instanceof SetStatusException) {
                                SetStatusException ex = (SetStatusException) throwable;
                                LOGGER.info("Handling known error {} for {}", exchange.getRequest().getURI(), ex.toString());
                                exchange.getResponse().setRawStatusCode(ex.getStatus());
                            } else {
                                LOGGER.error("Handling unknown error {}", exchange.getRequest().getURI(), throwable);
                            }
                            return chain.filter(exchange);
                        });
            } else {
                return chain.filter(exchange);
            }

        }

        private boolean isSecuredPath(ServerWebExchange exchange) {
            String url = exchange.getRequest().getPath().value();
            return url.startsWith("/chat");
        }

        private boolean isAaa(ServerWebExchange exchange) {
            Route route = exchange.getAttribute(GATEWAY_ROUTE_ATTR);
            return route != null && "aaa".equals(route.getId());
        }

        private Optional<String> getSessionCookie(MultiValueMap<String, HttpCookie> cookies) {
            HttpCookie session = cookies.getFirst(SESSION_COOKIE);
            if (session == null) {
                return Optional.empty();
            }
            return Optional.ofNullable(session.getValue());
        }

        @Override
        public int getOrder() {
            return Ordered.LOWEST_PRECEDENCE - 1;
        }
    }

}
