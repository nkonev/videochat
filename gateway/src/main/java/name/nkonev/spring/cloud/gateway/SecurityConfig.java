package name.nkonev.spring.cloud.gateway;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.context.annotation.Bean;
import org.springframework.core.Ordered;
import org.springframework.security.config.annotation.web.reactive.EnableWebFluxSecurity;
import org.springframework.security.config.web.server.ServerHttpSecurity;
import org.springframework.security.core.userdetails.MapReactiveUserDetailsService;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.web.server.SecurityWebFilterChain;
import org.springframework.session.data.redis.config.annotation.web.server.EnableRedisWebSession;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebSession;
import reactor.core.publisher.Mono;
import reactor.util.function.Tuple2;

import java.security.Principal;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;

@EnableRedisWebSession
@EnableWebFluxSecurity
public class SecurityConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityConfig.class);

    @Bean
    public SecurityWebFilterChain securityWebFilterChain(ServerHttpSecurity http) {
        return http.authorizeExchange()
                .pathMatchers("/", "/public/**").permitAll()
                .pathMatchers("/api/**").authenticated()
                .and().formLogin()
                .and().build();
    }

    @Bean
    public MapReactiveUserDetailsService userDetailsService() {
        var jlong = User.withDefaultPasswordEncoder().username("jlong").password("pw").roles("USER").build();
        var rwinch = User.withDefaultPasswordEncoder().username("rwinch").password("pw").roles("ADMIN", "USER").build();
        return new MapReactiveUserDetailsService(jlong, rwinch);
    }

    @Bean
    public InsertAuthHeadersFilter headerInserter() {
        return new InsertAuthHeadersFilter();
    }

    // inserted before NettyRoutingFilter which containing http client
    public static class InsertAuthHeadersFilter implements GlobalFilter, Ordered {

        private static final DateTimeFormatter DATE_TIME_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss Z")
                .withZone(ZoneId.systemDefault());
        private static final String X_AUTH_USERNAME = "X-Auth-Username";
        private static final String X_AUTH_SUBJECT = "X-Auth-Subject";
        private static final String X_AUTH_EXPIRESIN = "X-Auth-Expiresin";

        @Override
        public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
            Mono<Principal> principalMono = exchange.getPrincipal();
            Mono<WebSession> webSessionMono = exchange.getSession();
            Mono<Tuple2<Principal, WebSession>> tuple2Mono = principalMono.zipWith(webSessionMono);

            return tuple2Mono.flatMap(tuple2 -> {
                Principal principal = tuple2.getT1();
                WebSession session = tuple2.getT2();
                Instant creationTime = session.getCreationTime();
                Duration maxIdleTime = session.getMaxIdleTime();
                Instant expiresIn = creationTime.plus(maxIdleTime);
                String expiresInString = DATE_TIME_FORMATTER.format(expiresIn);

                ServerWebExchange modifiedExchange = exchange.mutate().request(builder -> {
                    builder.header(X_AUTH_USERNAME, principal.getName());
                    builder.header(X_AUTH_SUBJECT, principal.getName());
                    builder.header(X_AUTH_EXPIRESIN, expiresInString);
                }).build();
                LOGGER.info("{} '{}' inserting {}='{}', {}='{}', {}='{}'",
                        modifiedExchange.getRequest().getMethod(), modifiedExchange.getRequest().getURI(),
                        X_AUTH_USERNAME, principal.getName(),
                        X_AUTH_SUBJECT, principal.getName(),
                        X_AUTH_EXPIRESIN, expiresInString
                );
                return chain.filter(modifiedExchange);
            })
            // prevent leak when there aren't either session or principal - we always should invoke chain.filter(exchange) for close netty buffers
            .switchIfEmpty(chain.filter(exchange));
        }

        @Override
        public int getOrder() {
            return Ordered.LOWEST_PRECEDENCE-1;
        }
    }

}
