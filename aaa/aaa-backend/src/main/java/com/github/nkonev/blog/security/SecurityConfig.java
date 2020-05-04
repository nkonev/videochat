package com.github.nkonev.blog.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.dto.UserRole;
import com.github.nkonev.blog.security.checks.BlogPostAuthenticationChecks;
import com.github.nkonev.blog.security.checks.BlogPreAuthenticationChecks;
import com.github.nkonev.blog.security.converter.BearerOAuth2AccessTokenResponseConverter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.actuate.autoconfigure.security.servlet.EndpointRequest;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.converter.FormHttpMessageConverter;
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
import org.springframework.security.oauth2.client.registration.InMemoryClientRegistrationRepository;
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
    private BlogUserDetailsService blogUserDetailsService;

    @Autowired
    private BlogPreAuthenticationChecks blogPreAuthenticationChecks;

    @Autowired
    private BlogPostAuthenticationChecks blogPostAuthenticationChecks;

    @Autowired
    private BlogOAuth2UserService blogOAuth2UserService;

    @Autowired
    InMemoryClientRegistrationRepository clientRegistrationRepository;

    @Autowired
    private OAuth2ExceptionHandler OAuth2ExceptionHandler;

    @Autowired
    private NoOpAuthorizedClientRepository noOpAuthorizedClientRepository;


    @Bean
    public CsrfTokenRepository csrfTokenRepository() {
        return CookieCsrfTokenRepository.withHttpOnlyFalse();
    }

    @Bean
    public RESTAuthenticationLogoutSuccessHandler restAuthenticationLogoutSuccessHandler(ObjectMapper objectMapper) {
        return new RESTAuthenticationLogoutSuccessHandler(csrfTokenRepository(), objectMapper);
    }

    @Override
    protected void configure(AuthenticationManagerBuilder auth) throws Exception {
        // https://dzone.com/articles/spring-security-4-authenticate-and-authorize-users
        // http://www.programming-free.com/2015/09/spring-security-password-encryption.html
        auth.authenticationProvider(authenticationProvider());
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
                                userInfoEndpoint.userService(blogOAuth2UserService)
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

        http.headers().frameOptions().disable();
        http.headers().cacheControl().disable(); // see also com.github.nkonev.blog.controllers.AbstractImageUploadController#shouldReturnLikeCache
    }

    OAuth2AccessTokenResponseClient<OAuth2AuthorizationCodeGrantRequest> accessTokenResponseClient() {
        OAuth2AccessTokenResponseHttpMessageConverter oAuth2AccessTokenResponseHttpMessageConverter = new OAuth2AccessTokenResponseHttpMessageConverter();
        oAuth2AccessTokenResponseHttpMessageConverter.setTokenResponseConverter(new BearerOAuth2AccessTokenResponseConverter());
        RestTemplate restTemplate = new RestTemplate(Arrays.asList(
                new FormHttpMessageConverter(),
                oAuth2AccessTokenResponseHttpMessageConverter
        ));

        restTemplate.setErrorHandler(new OAuth2ErrorResponseErrorHandler());
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
    public DaoAuthenticationProvider authenticationProvider() {
        DaoAuthenticationProvider authenticationProvider = new DaoAuthenticationProvider();
        authenticationProvider.setUserDetailsService(blogUserDetailsService);
        authenticationProvider.setPasswordEncoder(passwordEncoder());
        authenticationProvider.setPreAuthenticationChecks(blogPreAuthenticationChecks);
        authenticationProvider.setPostAuthenticationChecks(blogPostAuthenticationChecks);
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
