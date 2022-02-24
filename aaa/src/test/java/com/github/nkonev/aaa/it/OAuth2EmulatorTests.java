package com.github.nkonev.aaa.it;

import com.github.nkonev.aaa.AbstractTestRunner;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.nimbusds.jose.JOSEException;
import com.nimbusds.jose.JWSAlgorithm;
import com.nimbusds.jose.JWSHeader;
import com.nimbusds.jose.JWSSigner;
import com.nimbusds.jose.crypto.RSASSASigner;
import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.KeyUse;
import com.nimbusds.jose.jwk.RSAKey;
import com.nimbusds.jwt.JWTClaimsSet;
import com.nimbusds.jwt.SignedJWT;
import io.netty.handler.codec.http.HttpHeaderNames;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.mockserver.integration.ClientAndServer;
import org.mockserver.model.Header;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;

import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPublicKey;
import java.util.Arrays;
import java.util.Date;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Supplier;

import static org.mockserver.integration.ClientAndServer.startClientAndServer;
import static org.mockserver.model.HttpRequest.request;
import static org.mockserver.model.HttpResponse.response;

public abstract class OAuth2EmulatorTests extends AbstractTestRunner {
    private static final int MOCK_SERVER_FACEBOOK_PORT = 10080;
    private static final int MOCK_SERVER_VKONTAKTE_PORT = 10081;
    private static final int MOCK_SERVER_GOOGLE_PORT = 10082;

    private static ClientAndServer mockServerFacebook;
    private static ClientAndServer mockServerVkontakte;
    private static ClientAndServer mockServerGoogle;

    @Autowired
    protected UserAccountRepository userAccountRepository;

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(OAuth2EmulatorTests.class);

    @BeforeAll
    public static void setUpClass() {
        LOGGER.info("Starting mock OAuth2 servers on ports {}", Arrays.asList(MOCK_SERVER_FACEBOOK_PORT, MOCK_SERVER_VKONTAKTE_PORT, MOCK_SERVER_GOOGLE_PORT));
        mockServerFacebook = startClientAndServer(MOCK_SERVER_FACEBOOK_PORT);
        mockServerVkontakte = startClientAndServer(MOCK_SERVER_VKONTAKTE_PORT);
        mockServerGoogle = startClientAndServer(MOCK_SERVER_GOOGLE_PORT);
    }

    @AfterAll
    public static void tearDownClass() throws Exception {
        LOGGER.info("Stopping mock OAuth2 servers");
        mockServerFacebook.stop();
        mockServerVkontakte.stop();
        mockServerGoogle.stop();
    }

    public static final String facebookLogin = "Nikita K";
    public static final String facebookId = "1234";
    public static final String vkontakteFirstName = "Никита";
    public static final String vkontakteLastName = "Конев";
    public static final String vkontakteLogin =vkontakteFirstName +  " " + vkontakteLastName;
    public static final String vkontakteId = "1212";
    public static final String googleLogin = "NIKITA KONEV";
    public static final String googleId = "1234567890";
    public static final String keycloakLogin = "user1";
    public static final String keycloakPassword = "user_password";
    public static final String keycloakId = "b5d67207-0996-4af0-bcb5-eee814687b30";

    @BeforeEach
    public void configureFacebookEmulator() throws InterruptedException {
        mockServerFacebook
                .when(request().withPath("/mock/facebook/dialog/oauth")).respond(httpRequest -> {
            String state = httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "state".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue();
            return response().withHeaders(
                    new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "text/html; charset=\"utf-8\""),
                    new Header(HttpHeaderNames.LOCATION.toString(), urlPrefix+"/api/login/oauth2/code/facebook?code=fake_code&state="+state)
            ).withStatusCode(302);
        });

        mockServerFacebook
                .when(request().withPath("/mock/facebook/oauth/access_token"))
                .respond(response().withHeaders(
                        new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                        ).withStatusCode(200).withBody("{\n" +
                                "  \"access_token\": \"fake-access-token\", \n" +
                                "  \"token_type\": \"bearer\",\n" +
                                "  \"expires_in\":  3600\n" +
                                "}")
                );

        mockServerFacebook
                .when(request().withPath("/mock/facebook/me"))
                .respond(response().withHeaders(
                        new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                        ).withStatusCode(200).withBody("{\n" +
                                "  \"id\": \""+facebookId+"\", \n" +
                                "  \"name\": \""+facebookLogin+"\",\n" +
                                "  \"picture\": {\n" +
                                "      \"data\": {\t\n" +
                                "           \"url\": \"http://localhost:9080/ava.png\"\n" +
                                "        }\n" +
                                "    }"+
                                "}")
                );

        userAccountRepository.findByUsername(facebookLogin).ifPresent(userAccount -> {
            userAccount = userAccount.withLocked(false);
            userAccount = userAccountRepository.save(userAccount);
        });

    }

    @BeforeEach
    public void clearOauthBindingsInDb() {
        final var deleteUsersSql = "DELETE FROM users WHERE username = :username";
        namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", facebookLogin));
        namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", vkontakteLogin));
        namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", googleLogin));

        namedParameterJdbcTemplate.update("UPDATE users SET vkontakte_id=NULL, facebook_id=NULL, google_id=NULL", Map.of());
    }

    @BeforeEach
    public void configureVkontakteEmulator(){
        mockServerVkontakte
                .when(request().withPath("/mock/vkontakte/authorize")).respond(httpRequest -> {
            String state = httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "state".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue();
            return response().withHeaders(
                    new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "text/html; charset=\"utf-8\""),
                    new Header(HttpHeaderNames.LOCATION.toString(), urlPrefix+"/api/login/oauth2/code/vkontakte?code=fake_code&state="+state)
            ).withStatusCode(302);
        });

        mockServerVkontakte
                .when(request().withPath("/mock/vkontakte/access_token"))
                .respond(response().withHeaders(
                        new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                        ).withStatusCode(200).withBody("{\n" +
                                "  \"access_token\": \"fake-access-token\", \n" +
                                "  \"token_type\": \"bearer\",\n" +
                                "  \"expires_in\":  3600\n" +
                                "}")
                );

        mockServerVkontakte
                .when(request().withPath("/mock/vkontakte/method/users.get"))
                .respond(response().withHeaders(
                        new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json; charset=\"utf-8\"")
                        ).withStatusCode(200).withBody("{\"response\": [{\"id\": "+vkontakteId+", \"first_name\": \""+vkontakteFirstName+"\", \"last_name\": \""+vkontakteLastName+"\"}]}")
                );

        userAccountRepository.findByUsername(vkontakteLogin).ifPresent(userAccount -> {
            userAccount = userAccount.withLocked(false);
            userAccountRepository.save(userAccount);
        });
    }

    private static KeyPair getKeyPair() throws NoSuchAlgorithmException {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("RSA");
        kpg.initialize(2048);
        return kpg.genKeyPair();
    }

    private static JWKSet jwkSet(KeyPair kp) {
        RSAKey.Builder builder = new RSAKey.Builder((RSAPublicKey)kp.getPublic())
                .keyUse(KeyUse.SIGNATURE)
                .algorithm(JWSAlgorithm.RS256)
                .keyID("fake-google-key-id");
        return new JWKSet(builder.build());
    }

    private static KeyPair keyPair;
    private static JWSSigner signer;
    private static JWKSet jwkSet;

    // aaa caches public keys, so in order to survive them across recreating mockservers they are put into static block
    static {
        try {
            keyPair = getKeyPair();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
        signer = new RSASSASigner(keyPair.getPrivate());
        jwkSet = jwkSet(keyPair);
    }

    @BeforeEach
    public void configureGoogleEmulator() throws NoSuchAlgorithmException {
        AtomicReference<String> nonceHolder = new AtomicReference<>();

        Supplier<String> tokenCreator = () -> {
            try {
                var currDate = new Date();
                JWTClaimsSet claimsSet = new JWTClaimsSet.Builder()
                        .subject(googleId)
                        .issuer("https://accounts.google.com")
                        .issueTime(currDate)
                        .expirationTime(new Date(currDate.getTime() + 600 * 1000))
                        .audience("987654321") // clientId
                        .claim("name", googleLogin)
                        .claim("admin", true)
                        .claim("nonce", nonceHolder.get())
                        .build();
                SignedJWT signedJWT = new SignedJWT(new JWSHeader(JWSAlgorithm.RS256), claimsSet);
                signedJWT.sign(signer);
                return signedJWT.serialize();
            } catch (JOSEException e) {
                throw new RuntimeException(e);
            }
        };

        mockServerGoogle
                .when(request().withPath("/mock/google/o/oauth2/v2/auth")).respond(httpRequest -> {
            String state = httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "state".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue();
            nonceHolder.set(httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "nonce".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue());
            return response().withHeaders(
                    new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "text/html; charset=\"utf-8\""),
                    new Header(HttpHeaderNames.LOCATION.toString(), urlPrefix+"/api/login/oauth2/code/google?code=fake_code&state="+state)
            ).withStatusCode(302);
        });

        // https://www.baeldung.com/spring-security-oauth2-jws-jwk
        mockServerGoogle
                .when(request().withPath("/mock/google/jwks")).respond(httpRequest -> {
                    return response().withHeaders(
                            new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                    ).withBody(jwkSet.toString())
                            .withStatusCode(200);
                });

        mockServerGoogle
                .when(request().withPath("/mock/google/oauth2/v4/token"))
                .respond(httpRequest -> {
                    return response().withHeaders(
                            new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                    ).withStatusCode(200).withBody("{\n" +
                            "  \"id_token\": \""+tokenCreator.get()+"\", \n" +
                            "  \"access_token\": \"fake-access-token\", \n" +
                            "  \"scope\": \"openid https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email\", \n" +
                            "  \"token_type\": \"Bearer\",\n" +
                            "  \"expires_in\":  3600\n" +
                            "}");
                });

        userAccountRepository.findByUsername(googleLogin).ifPresent(userAccount -> {
            userAccount = userAccount.withLocked(false);
            userAccountRepository.save(userAccount);
        });
    }



    @AfterEach
    public void resetFacebookEmulator(){
        mockServerFacebook.reset();
    }

    @AfterEach
    public void resetVkontakteEmulator(){
        mockServerVkontakte.reset();
    }

    @AfterEach
    public void resetGoogleEmulator(){
        mockServerGoogle.reset();
    }
}
