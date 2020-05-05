package name.nkonev.gateway;

import static org.springframework.cloud.gateway.support.ServerWebExchangeUtils.GATEWAY_ROUTE_ATTR;

import java.util.Optional;
import name.nkonev.users.UserServiceGrpc;
import name.nkonev.users.UserServiceGrpc.UserServiceBlockingStub;
import name.nkonev.users.UserSessionRequest;
import name.nkonev.users.UserSessionResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.cloud.gateway.route.Route;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.http.HttpCookie;
import org.springframework.util.MultiValueMap;
import org.springframework.web.server.ServerWebExchange;
import reactor.core.publisher.Mono;

@Configuration
public class SecurityConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityConfig.class);

    @Autowired
    private UserServiceGrpc.UserServiceBlockingStub userServiceStub;

    @Bean
    public SecurityFilter headerInserter() {
        return new SecurityFilter(userServiceStub);
    }

    // inserted before NettyRoutingFilter which containing http client
    public static class SecurityFilter implements GlobalFilter, Ordered {

        private final UserServiceGrpc.UserServiceBlockingStub userServiceStub;

        public static final String X_AUTH_USERNAME = "X-Auth-Username";
        public static final String X_AUTH_SUBJECT = "X-Auth-UserId";
        public static final String X_AUTH_EXPIRESIN = "X-Auth-Expiresin";

        public SecurityFilter(UserServiceBlockingStub userServiceStub) {
            this.userServiceStub = userServiceStub;
        }

        @Override
        public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
            Optional<String> sessionCookie = getSessionCookie(exchange.getRequest().getCookies());

            if (isSecuredPath(exchange) && !isAaa(exchange) && sessionCookie.isPresent()) {
                UserSessionRequest sessionRequest = UserSessionRequest.newBuilder()
                    .setSession(sessionCookie.get()).build();
                UserSessionResponse sessionResponse = userServiceStub.findBySession(sessionRequest);

                String username = sessionResponse.getUserName();
                long userid = sessionResponse.getUserId();
                long expiresIn = sessionResponse.getExpiresIn();
                
                ServerWebExchange modifiedExchange = exchange.mutate().request(builder -> {
                    builder.header(X_AUTH_USERNAME, username);
                    builder.header(X_AUTH_SUBJECT, "" + userid);
                    builder.header(X_AUTH_EXPIRESIN, ""+expiresIn);
                }).build();
                LOGGER.info("{} '{}' inserting {}='{}', {}='{}', {}='{}'",
                    modifiedExchange.getRequest().getMethod(),
                    modifiedExchange.getRequest().getURI(),
                    X_AUTH_USERNAME, username,
                    X_AUTH_SUBJECT, userid,
                    X_AUTH_EXPIRESIN, expiresIn
                );
                return chain.filter(modifiedExchange);
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
            return route !=null && "aaa".equals(route.getId());
        }

        private Optional<String> getSessionCookie(MultiValueMap<String, HttpCookie> cookies) {
            HttpCookie session = cookies.getFirst("SESSION");
            if (session == null) {return Optional.empty();}
            return Optional.ofNullable(session.getValue());
        }

        @Override
        public int getOrder() {
            return Ordered.LOWEST_PRECEDENCE-1;
        }
    }

}
