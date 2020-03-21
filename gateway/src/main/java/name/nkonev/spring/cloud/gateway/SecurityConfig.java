package name.nkonev.spring.cloud.gateway;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.cloud.gateway.filter.headers.HttpHeadersFilter;
import org.springframework.context.annotation.Bean;
import org.springframework.core.Ordered;
import org.springframework.http.HttpHeaders;
import org.springframework.security.authentication.AnonymousAuthenticationToken;
import org.springframework.security.config.annotation.web.reactive.EnableWebFluxSecurity;
import org.springframework.security.config.web.server.ServerHttpSecurity;
import org.springframework.security.core.authority.AuthorityUtils;
import org.springframework.security.core.userdetails.MapReactiveUserDetailsService;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.web.server.SecurityWebFilterChain;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebSession;
import reactor.core.publisher.Mono;
import reactor.util.function.Tuple2;

import java.security.Principal;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.time.temporal.ChronoField;

@EnableWebFluxSecurity
public class SecurityConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityConfig.class);

    @Bean
    public SecurityWebFilterChain securityWebFilterChain(
            ServerHttpSecurity http) {
        return http.authorizeExchange()
                .anyExchange().authenticated()
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
    public PreNettyRoutingFilter headerInserter() {
        return new PreNettyRoutingFilter();
    }

    public static class PreNettyRoutingFilter implements GlobalFilter, Ordered {

        DateTimeFormatter DATE_TIME_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss Z")
                .withZone(ZoneId.systemDefault());

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

                ServerWebExchange modifiedExchange = exchange.mutate().request(builder -> {
                    builder.header("X-Auth-Username", principal.getName());
                    builder.header("X-Auth-Subject", principal.getName());
                    builder.header("X-Auth-Expiresin", DATE_TIME_FORMATTER.format(expiresIn));
                }).build();
                return chain.filter(modifiedExchange);
            })
            // prevent leak when no session or principal - we always should invoke chain.filter(exchange) for close netty buffers
            .switchIfEmpty(chain.filter(exchange));
        }

        @Override
        public int getOrder() {
            return Ordered.LOWEST_PRECEDENCE-1;
        }
    }

}
