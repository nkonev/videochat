package com.github.nkonev.aaa.it;

import com.codeborne.selenide.Condition;
import com.codeborne.selenide.Selenide;
import com.github.nkonev.aaa.AbstractSeleniumRunner;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.FailoverUtils;
import com.github.nkonev.aaa.config.webdriver.Browser;
import com.github.nkonev.aaa.config.webdriver.SeleniumProperties;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.security.OAuth2Providers;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.password.PasswordEncoder;

import java.net.URI;

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

    private void clickGoogle() {
        $("#a-google").click();
    }

    private void clickLogout() {
        $("#btn-logout").click();
    }

    private class LoginPage {
        public LoginPage(String login, String password) {
            this.login = login;
            this.password = password;
        }

        private void openLoginPage() {
            open(urlPrefix+"/login.html");
        }

        private String login;
        private String password;

        private void login() {
            $("input#username").setValue(this.login);
            $("input#password").setValue(this.password);
            $("#btn-login").click();
        }
    }

    @Test
    public void testFacebookLogin()  {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        openOauth2TestPage();

        clickFacebook();
        $("#facebookId").should(Condition.appear, Condition.ownText(facebookId));

        UserAccount userAccount = FailoverUtils.retry(10, () -> userAccountRepository.findByUsername(facebookLogin).orElseThrow());
        Assertions.assertNotNull(userAccount.getId());
        Assertions.assertEquals(facebookLogin, userAccount.getUsername());
    }

    @Test
    public void testFacebookLoginAndMergeVkontakte()  {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        openOauth2TestPage();

        clickFacebook();
        $("#facebookId").should(Condition.appear, Condition.ownText(facebookId));

        UserAccount userAccount = FailoverUtils.retry(10, () -> userAccountRepository.findByUsername(facebookLogin).orElseThrow());
        Long facebookLoggedId = userAccount.getId();
        Assertions.assertNotNull(facebookLoggedId);
        Assertions.assertEquals(facebookLogin, userAccount.getUsername());
        String facebookId = userAccount.getOauth2Identifiers().getFacebookId();
        Assertions.assertNotNull(facebookId);
        Assertions.assertNull(userAccount.getOauth2Identifiers().getVkontakteId());
        long count = userAccountRepository.count();

        clickVkontakte();
        $("#vkontakteId").should(Condition.appear, Condition.ownText(vkontakteId));

        UserAccount userAccountFbAndVk = FailoverUtils.retry(10, () -> userAccountRepository.findByUsername(facebookLogin).orElseThrow());
        String userAccountFbAndVkFacebookId = userAccountFbAndVk.getOauth2Identifiers().getFacebookId();
        Assertions.assertNotNull(userAccountFbAndVkFacebookId);
        Assertions.assertNotNull(userAccountFbAndVk.getOauth2Identifiers().getVkontakteId());
        long countAfterVk = userAccountRepository.count();


        Assertions.assertEquals(facebookId, userAccountFbAndVkFacebookId);
        Assertions.assertEquals(count, countAfterVk);
        Assertions.assertEquals(userAccount.getUsername(), userAccountFbAndVk.getUsername());
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
        final String login600 = "generated_user_600";
        LoginPage loginPage = new LoginPage(login600, COMMON_PASSWORD);
        loginPage.openLoginPage();
        loginPage.login();
        UserAccount userAccount = userAccountRepository.findByUsername(login600).orElseThrow();
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // bind facebook to him
        openOauth2TestPage();
        clickFacebook();
        $("#facebookId").should(Condition.appear, Condition.ownText(facebookId));
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // logout
        clickLogout();

        // check that binding is preserved
        Selenide.refresh();
        // assert that he has facebook id
        UserAccount userAccountAfterBind = userAccountRepository.findByUsername(login600).orElseThrow();
        Assertions.assertNotNull(userAccountAfterBind.getOauth2Identifiers().getFacebookId());

        // login as another user to vk - vk id #1 save to database
        {
            openOauth2TestPage();
            clickVkontakte();

            Assertions.assertEquals(countInitial+1, userAccountRepository.count());
            // logout another user
            clickLogout();
        }

        // login facebook-bound user 600 again
        loginPage.openLoginPage();
        loginPage.login();
        // try to bind him vk, but emulator returns previous vk id #1 - here backend must argue that we already have vk id #1 in our database on another user
        openOauth2TestPage();
        clickVkontakte();
        Assertions.assertTrue($("body").has(Condition.text("Somebody already taken this vkontakte id")));
    }

    @Test
    public void checkUnbindFacebook() throws Exception {
        // login as 550
        final String login600 = "generated_user_550";
        LoginPage loginModal600 = new LoginPage(login600, COMMON_PASSWORD);
        loginModal600.openLoginPage();
        loginModal600.login();
        UserAccount userAccount = userAccountRepository.findByUsername(loginModal600.login).orElseThrow();

        // bind facebook
        openOauth2TestPage();
        clickFacebook();
        $("#facebookId").should(Condition.appear, Condition.ownText(facebookId));
        UserAccount userAccountAfterBindFacebook = userAccountRepository.findByUsername(loginModal600.login).orElseThrow();
        // assert facebook is bound - check database
        Assertions.assertNotNull(userAccountAfterBindFacebook.getOauth2Identifiers().getFacebookId());

        // logout
        clickLogout();

        // login again
        SessionHolder userAliceSession = login(loginModal600.login, loginModal600.password);

        final String FACEBOOK = "/" + OAuth2Providers.FACEBOOK;

        // unbind facebook
        RequestEntity myPostsRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ Constants.Urls.API+Constants.Urls.PROFILE+FACEBOOK))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(myPostsRequest1, String.class);
        Assertions.assertEquals(200, myPostsResponse1.getStatusCodeValue());

        // assert facebook is unbound - check database
        UserAccount userAccountAfterDeleteFacebook = userAccountRepository.findByUsername(loginModal600.login).orElseThrow();
        Assertions.assertNull(userAccountAfterDeleteFacebook.getOauth2Identifiers().getFacebookId());
    }



    @Test
    public void testGoogleLoginAndDelete() throws Exception {
        final String googlePassword = "dummy password";

        long countInitial = userAccountRepository.count();
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        openOauth2TestPage();

        clickGoogle();

        long countBeforeDelete = FailoverUtils.retry(10, () -> {
            long c = userAccountRepository.count();
            if (countInitial+1 != c) {
                throw new RuntimeException("User still not created");
            }
            return c;
        });

        UserAccount userAccount = userAccountRepository.findByUsername(googleLogin).orElseThrow();

        Assertions.assertNotNull(userAccount.getId());
        Assertions.assertNull(userAccount.getAvatar());
        Assertions.assertEquals(googleLogin, userAccount.getUsername());
        Assertions.assertEquals(googleId, userAccount.getOauth2Identifiers().getGoogleId());

        userAccount.setPassword(passwordEncoder.encode(googlePassword));
        userAccountRepository.save(userAccount);


        SessionHolder userNikitaSession = login(googleLogin, googlePassword);
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

}
