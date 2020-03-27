package name.nkonev.gateway;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.context.annotation.Bean;
import org.springframework.core.Ordered;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.server.reactive.ServerHttpResponse;
import org.springframework.security.config.annotation.web.reactive.EnableWebFluxSecurity;
import org.springframework.security.config.web.server.SecurityWebFiltersOrder;
import org.springframework.security.config.web.server.ServerHttpSecurity;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.server.SecurityWebFilterChain;
import org.springframework.security.web.server.ServerAuthenticationEntryPoint;
import org.springframework.security.web.server.ServerRedirectStrategy;
import org.springframework.security.web.server.authentication.RedirectServerAuthenticationSuccessHandler;
import org.springframework.security.web.server.authentication.logout.RedirectServerLogoutSuccessHandler;
import org.springframework.security.web.server.authentication.logout.ServerLogoutSuccessHandler;
import org.springframework.security.web.server.savedrequest.ServerRequestCache;
import org.springframework.security.web.server.savedrequest.WebSessionServerRequestCache;
import org.springframework.security.web.server.ui.LoginPageGeneratingWebFilter;
import org.springframework.security.web.server.ui.LogoutPageGeneratingWebFilter;
import org.springframework.security.web.server.util.matcher.*;
import org.springframework.session.data.redis.config.annotation.web.server.EnableRedisWebSession;
import org.springframework.util.Assert;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebSession;
import reactor.core.publisher.Mono;
import reactor.util.function.Tuple2;

import java.net.URI;
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
        LoginPageGeneratingWebFilter loginPageGeneratingWebFilter = new LoginPageGeneratingWebFilter();
        loginPageGeneratingWebFilter.setFormLoginEnabled(true);

        ServerWebExchangeMatcher get = ServerWebExchangeMatchers.pathMatchers(HttpMethod.GET, "/**");
        WebSessionServerRequestCache webSessionServerRequestCache = new WebSessionServerRequestCache();
        webSessionServerRequestCache.setSaveRequestMatcher(get);

        RedirectServerAuthenticationSuccessHandler authenticationSuccessHandler = new RedirectServerAuthenticationSuccessHandler();
        authenticationSuccessHandler.setRequestCache(webSessionServerRequestCache);
        authenticationSuccessHandler.setRedirectStrategy(new MyServerRedirectStrategy());
        return http.authorizeExchange()
                .pathMatchers("/", "/public/**"/*, "/login", "/logout"*/).permitAll()
                .pathMatchers("/api/**", "/self/**").authenticated()
                .and()
                .formLogin()
                .loginPage("/login")
                .authenticationEntryPoint(new MyEntryPoint(HttpStatus.UNAUTHORIZED, webSessionServerRequestCache))
                .authenticationSuccessHandler(authenticationSuccessHandler)
                .and()
                .logout()
                .logoutUrl("/logout")
//                .logoutSuccessHandler(logoutSuccessHandler("/bye"))
                .and()
                // restore default pages
                .addFilterAt(loginPageGeneratingWebFilter, SecurityWebFiltersOrder.LOGIN_PAGE_GENERATING)
                .addFilterAt(new LogoutPageGeneratingWebFilter(), SecurityWebFiltersOrder.LOGOUT_PAGE_GENERATING)
                .build();
    }
    public ServerLogoutSuccessHandler logoutSuccessHandler(String uri) {
        RedirectServerLogoutSuccessHandler successHandler = new RedirectServerLogoutSuccessHandler();
        successHandler.setLogoutSuccessUrl(URI.create(uri));
        return successHandler;
    }

    public static class MyEntryPoint implements ServerAuthenticationEntryPoint {

        final HttpStatus httpStatus;
        final ServerRequestCache requestCache;
        public MyEntryPoint(HttpStatus httpStatus, ServerRequestCache webSessionServerRequestCache) {
            this.httpStatus = httpStatus;
            this.requestCache = webSessionServerRequestCache;
        }

        @Override
        public Mono<Void> commence(ServerWebExchange exchange, AuthenticationException e) {
            return this.requestCache.saveRequest(exchange)
                    .then(Mono.fromRunnable(() -> exchange.getResponse().setStatusCode(this.httpStatus)));
        }
    }

    public static class MyServerRedirectStrategy implements ServerRedirectStrategy {
        private HttpStatus httpStatus = HttpStatus.FOUND;

        public Mono<Void> sendRedirect(ServerWebExchange exchange, URI location) {
            Assert.notNull(exchange, "exchange cannot be null");
            Assert.notNull(location, "location cannot be null");
            return Mono.fromRunnable(() -> {
                ServerHttpResponse response = exchange.getResponse();
                response.setStatusCode(this.httpStatus);
                response.getHeaders().setLocation(createLocation(exchange, location));
            });
        }

        private URI createLocation(ServerWebExchange exchange, URI location) {
            String url = location.toASCIIString();
            final String api = "/api";
            if (url.startsWith(api)) {
                String substring = url.substring(api.length());
                return URI.create(substring);
            }
            return location;
        }
    }

    @Bean
    public InsertAuthHeadersFilter headerInserter() {
        return new InsertAuthHeadersFilter();
    }

    // inserted before NettyRoutingFilter which containing http client
    public static class InsertAuthHeadersFilter implements GlobalFilter, Ordered {

        private static final DateTimeFormatter DATE_TIME_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss Z")
                .withZone(ZoneId.systemDefault());
        public static final String X_AUTH_USERNAME = "X-Auth-Username";
        public static final String X_AUTH_SUBJECT = "X-Auth-Subject";
        public static final String X_AUTH_EXPIRESIN = "X-Auth-Expiresin";

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
