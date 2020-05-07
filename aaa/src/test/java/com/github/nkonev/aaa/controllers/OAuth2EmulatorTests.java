/*package com.github.nkonev.aaa.controllers;

import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.services.UserDeleteService;
import com.github.nkonev.blog.webdriver.configuration.SeleniumConfiguration;
import io.netty.handler.codec.http.HttpHeaderNames;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.mockserver.integration.ClientAndServer;
import org.mockserver.model.Header;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;

import java.util.Collections;
import java.util.HashMap;

import static org.mockserver.integration.ClientAndServer.startClientAndServer;
import static org.mockserver.model.HttpRequest.request;
import static org.mockserver.model.HttpResponse.response;

public abstract class OAuth2EmulatorTests extends AbstractItTestRunner {
    private static final int MOCK_SERVER_FACEBOOK_PORT = 10080;
    private static final int MOCK_SERVER_VKONTAKTE_PORT = 10081;

    private static ClientAndServer mockServerFacebook;
    private static ClientAndServer mockServerVkontakte;

    @Autowired
    protected UserAccountRepository userAccountRepository;

    @Autowired
    protected SeleniumConfiguration seleniumConfiguration;

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    @Autowired
    protected UserDeleteService userDeleteService;

    @BeforeAll
    public static void setUpClass() {
        mockServerFacebook = startClientAndServer(MOCK_SERVER_FACEBOOK_PORT);
        mockServerVkontakte = startClientAndServer(MOCK_SERVER_VKONTAKTE_PORT);
    }

    @AfterAll
    public static void tearDownClass() throws Exception {
        mockServerFacebook.stop();
        mockServerVkontakte.stop();
    }

    public static final String facebookLogin = "Nikita K";
    public static final String vkontakteLogin = "Никита Конев";

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
                                "  \"id\": \"1234\", \n" +
                                "  \"name\": \""+facebookLogin+"\",\n" +
                                "  \"picture\": {\n" +
                                "      \"data\": {\t\n" +
                                "           \"url\": \"http://localhost:9080/ava.png\"\n" +
                                "        }\n" +
                                "    }"+
                                "}")
                );

        userAccountRepository.findByUsername(facebookLogin).ifPresent(userAccount -> {
            userAccount.setLocked(false);
            userAccount = userAccountRepository.save(userAccount);
        });

        clearOauthBindingsInDb();
    }

    private void clearOauthBindingsInDb() throws InterruptedException {
        String updatePosts = "UPDATE posts.post SET owner_id=(select id from auth.users WHERE username='deleted') WHERE owner_id = (select id from auth.users WHERE username=:username)";
        namedParameterJdbcTemplate.update(updatePosts, Collections.singletonMap("username", facebookLogin));
        namedParameterJdbcTemplate.update(updatePosts, Collections.singletonMap("username", vkontakteLogin));

        String updateComments = "UPDATE posts.comment SET owner_id=(select id from auth.users WHERE username='deleted') WHERE owner_id = (select id from auth.users WHERE username=:username)";
        namedParameterJdbcTemplate.update(updateComments, Collections.singletonMap("username", facebookLogin));
        namedParameterJdbcTemplate.update(updateComments, Collections.singletonMap("username", vkontakteLogin));

        String deleteUsers = "DELETE FROM auth.users WHERE username = :username";
        namedParameterJdbcTemplate.update(deleteUsers, Collections.singletonMap("username", facebookLogin));
        namedParameterJdbcTemplate.update(deleteUsers, Collections.singletonMap("username", vkontakteLogin));

        namedParameterJdbcTemplate.update("UPDATE auth.users SET vkontakte_id=NULL, facebook_id=NULL", new HashMap<>());
    }

    @AfterEach
    public void resetFacebookEmulator(){
        mockServerFacebook.reset();
        mockServerVkontakte.reset();
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
                        new Header(HttpHeaderNames.CONTENT_TYPE.toString(), "application/json")
                        ).withStatusCode(200).withBody("{\"response\": [{\"id\": 1212, \"first_name\": \"Никита\", \"last_name\": \"Конев\"}]}")
                );

        userAccountRepository.findByUsername(facebookLogin).ifPresent(userAccount -> {
            userAccount.setLocked(false);
            userAccountRepository.save(userAccount);
        });
    }

    @AfterEach
    public void resetVkontakteEmulator(){
        mockServerVkontakte.reset();
    }

}
*/