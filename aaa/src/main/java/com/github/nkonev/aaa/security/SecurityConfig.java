package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import com.github.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import com.github.nkonev.aaa.security.converter.BearerOAuth2AccessTokenResponseConverter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.actuate.autoconfigure.security.servlet.EndpointRequest;
import org.springframework.boot.web.client.RestTemplateBuilder;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.converter.FormHttpMessageConverter;
import org.springframework.security.authentication.AuthenticationProvider;
import org.springframework.security.authentication.dao.DaoAuthenticationProvider;
import org.springframework.security.config.annotation.authentication.builders.AuthenticationManagerBuilder;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configuration.WebSecurityConfigurerAdapter;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.oauth2.client.endpoint.DefaultAuthorizationCodeTokenResponseClient;
import org.springframework.security.oauth2.client.endpoint.OAuth2AccessTokenResponseClient;
import org.springframework.security.oauth2.client.endpoint.OAuth2AuthorizationCodeGrantRequest;
import org.springframework.security.oauth2.client.http.OAuth2ErrorResponseErrorHandler;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.core.http.converter.OAuth2AccessTokenResponseHttpMessageConverter;
import org.springframework.security.web.csrf.CookieCsrfTokenRepository;
import org.springframework.security.web.csrf.CsrfTokenRepository;
import org.springframework.web.client.RestTemplate;

import java.util.Arrays;

/**
 * http://websystique.com/springmvc/spring-mvc-4-and-spring-security-4-integration-example/
 * Created by nik on 08.06.17.
 */
@Configuration
@EnableWebSecurity
public class SecurityConfig extends WebSecurityConfigurerAdapter {

    public static final String API_LOGIN_URL = "/api/login";
    public static final String API_LOGOUT_URL = "/api/logout";

    public static final String USERNAME_PARAMETER = "username";
    public static final String PASSWORD_PARAMETER = "password";
    public static final String REMEMBER_ME_PARAMETER = "remember-me";

    public static final String API_LOGIN_OAUTH = "/api/login/oauth2";
    private static final String AUTHORIZATION_RESPONSE_BASE_URI = API_LOGIN_OAUTH + "/code/*";

    @Autowired
    private RESTAuthenticationEntryPoint authenticationEntryPoint;
    @Autowired
    private RESTAuthenticationFailureHandler authenticationFailureHandler;
    @Autowired
    private RESTAuthenticationSuccessHandler authenticationSuccessHandler;
    @Autowired
    private RESTAuthenticationLogoutSuccessHandler authenticationLogoutSuccessHandler;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private AuthenticationProvider ldapAuthenticationProvider;

    @Autowired
    private AaaPreAuthenticationChecks aaaPreAuthenticationChecks;

    @Autowired
    private AaaPostAuthenticationChecks aaaPostAuthenticationChecks;

    @Autowired
    private AaaOAuth2LoginUserService aaaOAuth2LoginUserService;

    @Autowired
    private AaaOAuth2AuthorizationCodeUserService aaaOAuth2AuthorizationCodeUserService;

    @Autowired
    ClientRegistrationRepository clientRegistrationRepository;

    @Autowired
    private OAuth2ExceptionHandler OAuth2ExceptionHandler;

    @Autowired
    private NoOpAuthorizedClientRepository noOpAuthorizedClientRepository;

    @Autowired
    private CustomConfig customConfig;

    @Value("${csrf.cookie.secure:false}")
    private boolean cookieSecure;

    @Value("${csrf.cookie.http-only:false}")
    private boolean cookieHttpOnly;

    @Value("${csrf.cookie.name:VIDEOCHAT_XSRF_TOKEN}")
    private String cookieName;

    @Bean
    public CsrfTokenRepository csrfTokenRepository() {
        final CookieCsrfTokenRepository cookieCsrfTokenRepository = new CookieCsrfTokenRepository();
        cookieCsrfTokenRepository.setCookieName(cookieName);
        cookieCsrfTokenRepository.setSecure(cookieSecure);
        cookieCsrfTokenRepository.setCookieHttpOnly(cookieHttpOnly);
        return cookieCsrfTokenRepository;
    }

    @Override
    protected void configure(AuthenticationManagerBuilder auth) throws Exception {
        // https://dzone.com/articles/spring-security-4-authenticate-and-authorize-users
        // http://www.programming-free.com/2015/09/spring-security-password-encryption.html
        auth.authenticationProvider(ldapAuthenticationProvider);
        auth.authenticationProvider(dbAuthenticationProvider());
    }

    @Bean
    public PasswordEncoder passwordEncoder() {
        return new BCryptPasswordEncoder(); // default strength is BCrypt.GENSALT_DEFAULT_LOG2_ROUNDS=10
    }

    @Override
    protected void configure(HttpSecurity http) throws Exception {
        http.authorizeRequests()
                .antMatchers("/favicon.ico", "/static/**", Constants.Urls.API+"/**").permitAll();
        http.authorizeRequests()
                .antMatchers(Constants.Urls.API+ Constants.Urls.ADMIN+"/**").hasAuthority(UserRole.ROLE_ADMIN.name());
        http.authorizeRequests().requestMatchers(EndpointRequest.toAnyEndpoint()).permitAll();

        http.csrf()
                .csrfTokenRepository(csrfTokenRepository());
        http.exceptionHandling()
                .authenticationEntryPoint(authenticationEntryPoint);

        http.formLogin()
                .loginPage(API_LOGIN_URL).usernameParameter(USERNAME_PARAMETER).passwordParameter(PASSWORD_PARAMETER).permitAll()
                .successHandler(authenticationSuccessHandler)
                .failureHandler(authenticationFailureHandler)

        .and().logout().logoutUrl(API_LOGOUT_URL).logoutSuccessHandler(authenticationLogoutSuccessHandler).permitAll();

        http.oauth2Login(oauth2Login ->
                oauth2Login
                        .authorizedClientRepository(noOpAuthorizedClientRepository)
                        .userInfoEndpoint(userInfoEndpoint ->
                                userInfoEndpoint.userService(aaaOAuth2LoginUserService)
                                        .oidcUserService(aaaOAuth2AuthorizationCodeUserService)
                        )
                        .authorizationEndpoint(authorizationEndpointConfig -> {
                            authorizationEndpointConfig.authorizationRequestResolver(oAuth2AuthorizationRequestResolver());
                            authorizationEndpointConfig.baseUri(API_LOGIN_OAUTH);
                        })

                        .successHandler(new OAuth2AuthenticationSuccessHandler())
                        .failureHandler(OAuth2ExceptionHandler)
                        .redirectionEndpoint(redirectionEndpointConfig -> redirectionEndpointConfig.baseUri(AUTHORIZATION_RESPONSE_BASE_URI))
                        .tokenEndpoint(tokenEndpointConfig -> {
                            tokenEndpointConfig.accessTokenResponseClient(this.accessTokenResponseClient());
                        })
        );

        http.headers().frameOptions().deny();
        http.headers().cacheControl().disable(); // see also AbstractImageUploadController#shouldReturnLikeCache
    }

    OAuth2AccessTokenResponseClient<OAuth2AuthorizationCodeGrantRequest> accessTokenResponseClient() {
        OAuth2AccessTokenResponseHttpMessageConverter oAuth2AccessTokenResponseHttpMessageConverter = new OAuth2AccessTokenResponseHttpMessageConverter();
        oAuth2AccessTokenResponseHttpMessageConverter.setAccessTokenResponseConverter(new BearerOAuth2AccessTokenResponseConverter());

        RestTemplate restTemplate = new RestTemplateBuilder()
                .setConnectTimeout(customConfig.getRestClientConnectTimeout())
                .setReadTimeout(customConfig.getRestClientReadTimeout())
                .messageConverters(Arrays.asList(
                        new FormHttpMessageConverter(),
                        oAuth2AccessTokenResponseHttpMessageConverter
                ))
                .errorHandler(new OAuth2ErrorResponseErrorHandler())
                .build();
        DefaultAuthorizationCodeTokenResponseClient defaultAuthorizationCodeTokenResponseClient = new DefaultAuthorizationCodeTokenResponseClient();
        defaultAuthorizationCodeTokenResponseClient.setRestOperations(restTemplate);
        return defaultAuthorizationCodeTokenResponseClient;
    }

    @Bean
    OAuth2AuthorizationRequestResolver oAuth2AuthorizationRequestResolver() {
        DefaultOAuth2AuthorizationRequestResolver defaultOAuth2AuthorizationRequestResolver = new DefaultOAuth2AuthorizationRequestResolver(clientRegistrationRepository, API_LOGIN_OAUTH);
        return new WithRefererInStateOAuth2AuthorizationRequestResolver(defaultOAuth2AuthorizationRequestResolver);
    }

    @Bean
    public DaoAuthenticationProvider dbAuthenticationProvider() {
        DaoAuthenticationProvider authenticationProvider = new DaoAuthenticationProvider();
        authenticationProvider.setUserDetailsService(aaaUserDetailsService);
        authenticationProvider.setPasswordEncoder(passwordEncoder());
        authenticationProvider.setPreAuthenticationChecks(aaaPreAuthenticationChecks);
        authenticationProvider.setPostAuthenticationChecks(aaaPostAuthenticationChecks);
        return authenticationProvider;
    }

//    @Bean
//    public PersistentTokenBasedRememberMeServices getPersistentTokenBasedRememberMeServices() {
//        PersistentTokenBasedRememberMeServices tokenBasedservice = new PersistentTokenBasedRememberMeServices(
//                REMEMBER_ME_PARAMETER, userDetailsService, tokenRepository);
//        return tokenBasedservice;
//    }

//    @Bean
//    public AuthenticationTrustResolver getAuthenticationTrustResolver() {
//        return new AuthenticationTrustResolverImpl();
//    }

}
