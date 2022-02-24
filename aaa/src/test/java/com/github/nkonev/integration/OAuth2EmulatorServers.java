package com.github.nkonev.integration;

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
import org.mockserver.integration.ClientAndServer;
import org.mockserver.model.Header;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPublicKey;
import java.util.Arrays;
import java.util.Date;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Supplier;

import static com.github.nkonev.aaa.it.OAuth2EmulatorTests.*;
import static org.mockserver.integration.ClientAndServer.startClientAndServer;
import static org.mockserver.model.HttpRequest.request;
import static org.mockserver.model.HttpResponse.response;

public class OAuth2EmulatorServers {
    private static final int MOCK_SERVER_FACEBOOK_PORT = 10080;
    private static final int MOCK_SERVER_VKONTAKTE_PORT = 10081;
    private static final int MOCK_SERVER_GOOGLE_PORT = 10082;

    private static ClientAndServer mockServerFacebook;
    private static ClientAndServer mockServerVkontakte;
    private static ClientAndServer mockServerGoogle;

    private static KeyPair keyPair;
    private static JWSSigner signer;
    private static JWKSet jwkSet;

    private static final Logger LOGGER = LoggerFactory.getLogger(OAuth2EmulatorServers.class);

    // aaa caches public keys, so in order to survive them across recreating mockservers they are put into static block
    static {
        try {
            LOGGER.info("Generating new RSA keypair (served in fake google jwks endpoint)");
            keyPair = getKeyPair();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
        signer = new RSASSASigner(keyPair.getPrivate());
        jwkSet = jwkSet(keyPair);
    }

    public static void start() {
        LOGGER.info("Starting mock OAuth2 servers on ports {}", Arrays.asList(MOCK_SERVER_FACEBOOK_PORT, MOCK_SERVER_VKONTAKTE_PORT, MOCK_SERVER_GOOGLE_PORT));
        mockServerFacebook = startClientAndServer(MOCK_SERVER_FACEBOOK_PORT);
        mockServerVkontakte = startClientAndServer(MOCK_SERVER_VKONTAKTE_PORT);
        mockServerGoogle = startClientAndServer(MOCK_SERVER_GOOGLE_PORT);
    }

    public static void stop() throws Exception {
        LOGGER.info("Stopping mock OAuth2 servers");
        mockServerFacebook.stop();
        mockServerVkontakte.stop();
        mockServerGoogle.stop();
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

    public static void resetFacebookEmulator(){
        mockServerFacebook.reset();
    }

    public static void resetVkontakteEmulator(){
        mockServerVkontakte.reset();
    }

    public static void resetGoogleEmulator(){
        mockServerGoogle.reset();
    }

    public static void configureFacebookEmulator(String baseUrl) {
        mockServerFacebook
                .when(request().withPath("/mock/facebook/dialog/oauth")).respond(httpRequest -> {
                    String state = httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "state".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue();
                    return response().withHeaders(
                            new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "text/html; charset=\"utf-8\""),
                            new Header(HttpHeaderNames.LOCATION.toString(), baseUrl+"/api/login/oauth2/code/facebook?code=fake_code&state="+state)
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
    }


    public static void configureVkontakteEmulator(String baseUrl){
        mockServerVkontakte
                .when(request().withPath("/mock/vkontakte/authorize")).respond(httpRequest -> {
                    String state = httpRequest.getQueryStringParameters().getEntries().stream().filter(parameter -> "state".equals(parameter.getName().getValue())).findFirst().get().getValues().get(0).getValue();
                    return response().withHeaders(
                            new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "text/html; charset=\"utf-8\""),
                            new Header(HttpHeaderNames.LOCATION.toString(), baseUrl+"/api/login/oauth2/code/vkontakte?code=fake_code&state="+state)
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
    }


    public static void configureGoogleEmulator(String baseUrl) {
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
                            new Header(HttpHeaderNames.LOCATION.toString(), baseUrl+"/api/login/oauth2/code/google?code=fake_code&state="+state)
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
    }

}
