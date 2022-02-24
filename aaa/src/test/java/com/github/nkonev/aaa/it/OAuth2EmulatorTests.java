package com.github.nkonev.aaa.it;

import com.github.nkonev.aaa.AbstractTestRunner;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.integration.OAuth2EmulatorServers;
import com.github.nkonev.integration.UserTestService;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.beans.factory.annotation.Autowired;

public abstract class OAuth2EmulatorTests extends AbstractTestRunner {

    @Autowired
    protected UserAccountRepository userAccountRepository;

    @Autowired
    private UserTestService userTestService;

    @BeforeAll
    public static void setUpClass() {
        OAuth2EmulatorServers.start();
    }

    @AfterAll
    public static void tearDownClass() throws Exception {
        OAuth2EmulatorServers.stop();
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
    public void clearOauthBindingsInDb() {
        userTestService.clearOauthBindingsInDb();
    }

    @BeforeEach
    public void configureFacebookEmulator() {
        OAuth2EmulatorServers.configureFacebookEmulator(urlPrefix);
    }

    @BeforeEach
    public void configureVkontakteEmulator(){
        OAuth2EmulatorServers.configureVkontakteEmulator(urlPrefix);
    }

    @BeforeEach
    public void configureGoogleEmulator() {
        OAuth2EmulatorServers.configureGoogleEmulator(urlPrefix);
    }


    @AfterEach
    public void resetFacebookEmulator(){
        OAuth2EmulatorServers.resetFacebookEmulator();
    }

    @AfterEach
    public void resetVkontakteEmulator(){
        OAuth2EmulatorServers.resetVkontakteEmulator();
    }

    @AfterEach
    public void resetGoogleEmulator(){
        OAuth2EmulatorServers.resetGoogleEmulator();
    }
}
