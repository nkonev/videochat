package name.nkonev.aaa.security;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import name.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import name.nkonev.aaa.security.converter.BearerOAuth2AccessTokenResponseConverter;
import name.nkonev.aaa.services.RefererService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.web.client.RestTemplateBuilder;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.JdkClientHttpRequestFactory;
import org.springframework.http.converter.FormHttpMessageConverter;
import org.springframework.security.authentication.dao.DaoAuthenticationProvider;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.oauth2.client.endpoint.DefaultAuthorizationCodeTokenResponseClient;
import org.springframework.security.oauth2.client.endpoint.OAuth2AccessTokenResponseClient;
import org.springframework.security.oauth2.client.endpoint.OAuth2AuthorizationCodeGrantRequest;
import org.springframework.security.oauth2.client.http.OAuth2ErrorResponseErrorHandler;
import org.springframework.security.oauth2.client.registration.ClientRegistrationRepository;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.core.http.converter.OAuth2AccessTokenResponseHttpMessageConverter;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.csrf.CookieCsrfTokenRepository;
import org.springframework.security.web.csrf.CsrfTokenRepository;
import org.springframework.security.web.csrf.CsrfTokenRequestAttributeHandler;
import org.springframework.web.client.RestTemplate;

import java.util.Arrays;

@Configuration
@EnableWebSecurity
public class SecurityConfig {

    public static final String API_LOGIN_URL = Constants.Urls.EXTERNAL_API + "/login";
    public static final String API_LOGOUT_URL = Constants.Urls.EXTERNAL_API + "/logout";

    public static final String USERNAME_PARAMETER = "username";
    public static final String PASSWORD_PARAMETER = "password";
    public static final String REMEMBER_ME_PARAMETER = "remember-me";

    public static final String API_LOGIN_OAUTH = Constants.Urls.EXTERNAL_API + "/login/oauth2";
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
    private LdapAuthenticationProvider ldapAuthenticationProvider;

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
    private OAuth2AuthenticationSuccessHandler oAuth2AuthenticationSuccessHandler;

    @Autowired
    private OAuth2ExceptionHandler oAuth2ExceptionHandler;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private RefererService refererService;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Bean
    public CsrfTokenRepository csrfTokenRepository() {
        final CookieCsrfTokenRepository cookieCsrfTokenRepository = new CookieCsrfTokenRepository();
        cookieCsrfTokenRepository.setCookieName(aaaProperties.csrf().cookie().name());
        cookieCsrfTokenRepository.setCookieCustomizer(responseCookieBuilder -> {
            responseCookieBuilder.sameSite(aaaProperties.csrf().cookie().sameSite());
            responseCookieBuilder.secure(aaaProperties.csrf().cookie().secure());
            responseCookieBuilder.httpOnly(aaaProperties.csrf().cookie().httpOnly());
        });
        return cookieCsrfTokenRepository;
    }

    // https://spring.io/blog/2022/02/21/spring-security-without-the-websecurityconfigureradapter
    @Bean
    public SecurityFilterChain securityFilterChain(HttpSecurity http) throws Exception {
        return http.authorizeHttpRequests(c -> {
            c.requestMatchers("/**").permitAll();
        }).csrf(c -> {
                var requestHandler = new CsrfTokenRequestAttributeHandler(); // disabling deferred needed in order not to fail the first request, it's seen from e2e-test
                requestHandler.setCsrfRequestAttributeName(null); // https://docs.spring.io/spring-security/reference/servlet/exploits/csrf.html#deferred-csrf-token
                c.csrfTokenRepository(csrfTokenRepository())
                .csrfTokenRequestHandler(requestHandler)
                .ignoringRequestMatchers(Constants.Urls.INTERNAL_API+ "/**");
        }).exceptionHandling(c -> {
            c.authenticationEntryPoint(authenticationEntryPoint);
        }).formLogin(c -> {
            c.loginPage(API_LOGIN_URL).usernameParameter(USERNAME_PARAMETER).passwordParameter(PASSWORD_PARAMETER).permitAll()
                .successHandler(authenticationSuccessHandler)
                .failureHandler(authenticationFailureHandler);
        }).logout(c -> {
            c.logoutUrl(API_LOGOUT_URL).logoutSuccessHandler(authenticationLogoutSuccessHandler).permitAll();
        }).oauth2Login(oauth2Login ->
                oauth2Login
                        .userInfoEndpoint(userInfoEndpoint ->
                                userInfoEndpoint.userService(aaaOAuth2LoginUserService)
                                        .oidcUserService(aaaOAuth2AuthorizationCodeUserService)
                        )
                        .authorizationEndpoint(authorizationEndpointConfig -> {
                            authorizationEndpointConfig.authorizationRequestResolver(oAuth2AuthorizationRequestResolver());
                            authorizationEndpointConfig.baseUri(API_LOGIN_OAUTH);
                        })

                        .successHandler(oAuth2AuthenticationSuccessHandler)
                        .failureHandler(oAuth2ExceptionHandler)
                        .redirectionEndpoint(redirectionEndpointConfig -> redirectionEndpointConfig.baseUri(AUTHORIZATION_RESPONSE_BASE_URI))
                        .tokenEndpoint(tokenEndpointConfig -> {
                            tokenEndpointConfig.accessTokenResponseClient(this.accessTokenResponseClient());
                        })
        )
        .authenticationProvider(ldapAuthenticationProvider)
        .authenticationProvider(dbAuthenticationProvider())
        .headers(c -> {
            c.frameOptions(fc -> {
                fc.deny();
            });
            c.cacheControl(cc -> {
                cc.disable(); // see also AbstractImageUploadController#shouldReturnLikeCache
            });
        })
        .build();
    }

    OAuth2AccessTokenResponseClient<OAuth2AuthorizationCodeGrantRequest> accessTokenResponseClient() {
        OAuth2AccessTokenResponseHttpMessageConverter oAuth2AccessTokenResponseHttpMessageConverter = new OAuth2AccessTokenResponseHttpMessageConverter();
        oAuth2AccessTokenResponseHttpMessageConverter.setAccessTokenResponseConverter(new BearerOAuth2AccessTokenResponseConverter());

        RestTemplate restTemplate = new RestTemplateBuilder()
                .connectTimeout(aaaProperties.httpClient().connectTimeout())
                .readTimeout(aaaProperties.httpClient().readTimeout())
                .requestFactory(JdkClientHttpRequestFactory.class)
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
        return new WithRefererInSessionOAuth2AuthorizationRequestResolver(defaultOAuth2AuthorizationRequestResolver, refererService);
    }

    @Bean
    public DaoAuthenticationProvider dbAuthenticationProvider() {
        DaoAuthenticationProvider authenticationProvider = new DaoAuthenticationProvider();
        authenticationProvider.setUserDetailsService(aaaUserDetailsService);
        authenticationProvider.setPasswordEncoder(passwordEncoder);
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
