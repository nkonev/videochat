package com.github.nkonev.aaa.it;

import com.codeborne.selenide.Condition;
import com.codeborne.selenide.Selenide;
import com.github.nkonev.aaa.AbstractSeleniumRunner;
import com.github.nkonev.aaa.CommonTestConstants;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.FailoverUtils;
import com.github.nkonev.aaa.config.webdriver.Browser;
import com.github.nkonev.aaa.config.webdriver.SeleniumProperties;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.security.crypto.password.PasswordEncoder;

import java.net.URI;
import java.time.LocalDateTime;

import static com.codeborne.selenide.Selenide.$;
import static com.codeborne.selenide.Selenide.open;
import static com.github.nkonev.aaa.CommonTestConstants.COMMON_PASSWORD;
import static com.github.nkonev.aaa.CommonTestConstants.HEADER_XSRF_TOKEN;
import static com.github.nkonev.aaa.Constants.Urls.API;
import static org.springframework.http.HttpHeaders.COOKIE;


public class UserProfileOauth2Test extends AbstractSeleniumRunner {

    @Autowired
    private SeleniumProperties seleniumConfiguration;

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    @Autowired
    private PasswordEncoder passwordEncoder;

    private void openOauth2TestPage() {
        open(urlPrefix+"/oauth2.html");
    }

    private void clickFacebook() {
        $("#a-facebook").click();
    }

    private void clickVkontakte() {
        $("#a-vkontakte").click();
    }

    private void clickLogout() {
        $("#btn-logout").click();
    }

    private void openLoginPage() {
        open(urlPrefix+"/login.html");
    }

    @Test
    public void testFacebookLogin()  {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        openOauth2TestPage();

        clickFacebook();

        UserAccount userAccount = FailoverUtils.retry(10, () -> userAccountRepository.findByUsername(facebookLogin).orElseThrow());
        Assertions.assertNotNull(userAccount.getId());
        Assertions.assertNotNull(userAccount.getAvatar());
        Assertions.assertTrue(userAccount.getAvatar().startsWith("/"));
        Assertions.assertEquals(facebookLogin, userAccount.getUsername());
    }


    @Test
    public void testVkontakteLoginAndDelete() throws Exception {
        final String vkontaktePassword = "dummy password";

        long countInitial = userAccountRepository.count();
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        openOauth2TestPage();

        clickVkontakte();

        long countBeforeDelete = FailoverUtils.retry(10, () -> {
            long c = userAccountRepository.count();
            if (countInitial+1 != c) {
                throw new RuntimeException("User still not created");
            }
            return c;
        });

        UserAccount userAccount = userAccountRepository.findByUsername(vkontakteLogin).orElseThrow();

        Assertions.assertNotNull(userAccount.getId());
        Assertions.assertNull(userAccount.getAvatar());
        Assertions.assertEquals(vkontakteLogin, userAccount.getUsername());

        userAccount.setPassword(passwordEncoder.encode(vkontaktePassword));
        userAccountRepository.save(userAccount);


        SessionHolder userNikitaSession = login(vkontakteLogin, vkontaktePassword);
        RequestEntity selfDeleteRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userNikitaSession.newXsrf)
                .header(COOKIE, userNikitaSession.getCookiesArray())
                .build();
        ResponseEntity<String> selfDeleteResponse1 = testRestTemplate.exchange(selfDeleteRequest1, String.class);
        Assertions.assertEquals(200, selfDeleteResponse1.getStatusCodeValue());

        FailoverUtils.retry(10, () -> {
            long countAfter = userAccountRepository.count();
            Assertions.assertEquals(countBeforeDelete-1, countAfter);
            return null;
        });
    }

    @Test
    public void testBindIdToAccountAndConflict() throws Exception {
        long countInitial = userAccountRepository.count();

        // login as regular user 600
        openLoginPage();
        final String login600 = "generated_user_600";
        $("input#username").setValue(login600);
        $("input#password").setValue(COMMON_PASSWORD);
        $("#btn-login").click();
        UserAccount userAccount = userAccountRepository.findByUsername(login600).orElseThrow();
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // bind facebook to him
        openOauth2TestPage();
        clickFacebook();
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // logout
        clickLogout();

        // check that binding is preserved
        Selenide.refresh();
        // assert that he has facebook id
        UserAccount userAccountAfterBind = userAccountRepository.findByUsername(login600).orElseThrow();
        Assertions.assertNotNull(userAccountAfterBind.getOauthIdentifiers().getFacebookId());

        // login as another user to vo - vk id #1 save to database
        {
            openOauth2TestPage();
            clickVkontakte();

            Assertions.assertEquals(countInitial+1, userAccountRepository.count());
            // logout another user
            clickLogout();
        }

        // login facebook-bound user 600 again
        openLoginPage();
        $("input#username").setValue(login600);
        $("input#password").setValue(COMMON_PASSWORD);
        $("#btn-login").click();
        // try to bind him vk, but emulator returns previous vk id #1 - here backend must argue that we already have vk id #1 in our database on another user
        openOauth2TestPage();
        clickVkontakte();
        Assertions.assertTrue($("body").has(Condition.text("Somebody already taken this vkontakte id")));
    }

    /*@Test
    public void checkUnbindFacebook() throws Exception {
        IndexPage indexPage = new IndexPage(urlPrefix);
        indexPage.openPage();

        UserProfilePage userPage = new UserProfilePage(urlPrefix, driver);
        final String login600 = "generated_user_550";

        LoginModal loginModal600 = new LoginModal(login600, COMMON_PASSWORD);
        loginModal600.openLoginModal();
        loginModal600.login();
        UserAccount userAccount = userAccountRepository.findByUsername(login600).orElseThrow();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertThisIsYou();
        userPage.edit();

        userPage.bindFacebook();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertHasFacebook();
        Selenide.refresh();
        userPage.assertHasFacebook();

        loginModal600.logout();

        loginModal600.openLoginModal();
        loginModal600.login();

        userPage.openPage(userAccount.getId().intValue());
        userPage.assertThisIsYou();
        userPage.edit();

        userPage.unBindFacebook();
        userPage.assertNotHasFacebook();
        Selenide.refresh();
        loginModal600.logout();
        Selenide.refresh();
        loginModal600.openLoginModal();
        loginModal600.login();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertThisIsYou();
        userPage.assertNotHasFacebook();

    }*/
}
