package name.nkonev.gateway;

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
import org.springframework.context.annotation.Bean;
import org.springframework.core.Ordered;
import org.springframework.http.HttpCookie;
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
import org.springframework.util.MultiValueMap;
import org.springframework.web.server.ServerWebExchange;
import reactor.core.publisher.Mono;

import java.net.URI;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;

//@EnableRedisWebSession
@EnableWebFluxSecurity
public class SecurityConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(SecurityConfig.class);

    @Autowired
    private UserServiceGrpc.UserServiceBlockingStub userServiceStub;

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
        return new InsertAuthHeadersFilter(userServiceStub);
    }

    // inserted before NettyRoutingFilter which containing http client
    public static class InsertAuthHeadersFilter implements GlobalFilter, Ordered {

        private final UserServiceGrpc.UserServiceBlockingStub userServiceStub;

        private static final DateTimeFormatter DATE_TIME_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss Z")
                .withZone(ZoneId.systemDefault());
        public static final String X_AUTH_USERNAME = "X-Auth-Username";
        public static final String X_AUTH_SUBJECT = "X-Auth-UserId";
        public static final String X_AUTH_EXPIRESIN = "X-Auth-Expiresin";

        public InsertAuthHeadersFilter(
            UserServiceBlockingStub userServiceStub) {
            this.userServiceStub = userServiceStub;
        }

        @Override
        public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
            Optional<String> sessionCookie = getSessionCookie(exchange.getRequest().getCookies());

            if (sessionCookie.isPresent()) {
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
