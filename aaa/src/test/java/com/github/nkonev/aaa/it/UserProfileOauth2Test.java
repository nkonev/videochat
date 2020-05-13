package com.github.nkonev.aaa.it;

import com.github.nkonev.aaa.AbstractSeleniumRunner;
import com.github.nkonev.aaa.config.webdriver.Browser;
import com.github.nkonev.aaa.config.webdriver.SeleniumProperties;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;

import static com.codeborne.selenide.Selenide.$;
import static com.codeborne.selenide.Selenide.open;


public class UserProfileOauth2Test extends AbstractSeleniumRunner {

    @Autowired
    private SeleniumProperties seleniumConfiguration;

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    @Test
    public void testFacebookLogin()  {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        open(urlPrefix+"/oauth2.html");

        $("#a-facebook").click();

        UserAccount userAccount = userAccountRepository.findByUsername(facebookLogin).orElseThrow();
        Assertions.assertNotNull(userAccount.getId());
        Assertions.assertNotNull(userAccount.getAvatar());
        Assertions.assertTrue(userAccount.getAvatar().startsWith("/"));
        Assertions.assertEquals("Nikita K", userAccount.getUsername());
    }

    /*@Test
    public void testVkontakteLoginAndDelete() throws Exception {
        long countInitial = userAccountRepository.count();
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        IndexPage indexPage = new IndexPage(urlPrefix);
        indexPage.openPage();

        LoginModal loginModal = new LoginModal();
        loginModal.openLoginModal();
        loginModal.loginVkontakte();

        Assertions.assertEquals(vkontakteLogin, UserNav.getLogin());

        long countBefore = userAccountRepository.count();
        Assertions.assertEquals(countInitial+1, countBefore);

        // now we attempt to change email
        UserProfilePage userPage = new UserProfilePage(urlPrefix, driver);
        UserAccount userAccount = userAccountRepository.findByUsername(vkontakteLogin).orElseThrow();
        userPage.openPage(userAccount.getId().intValue());
        LocalDateTime lastLoginFirst = userAccount.getLastLoginDateTime();
        userPage.assertThisIsYou();
        userPage.edit();
        userPage.setEmail("new-email-for-vkontakte-user@gmail.not");
        userPage.save();
        userPage.assertEmail("new-email-for-vkontakte-user@gmail.not");

        loginModal.logout();

        loginModal.openLoginModal();
        loginModal.loginVkontakte();

        UserAccount userAccountUpdated = userAccountRepository.findByUsername(vkontakteLogin).orElseThrow();
        LocalDateTime lastLoginSecond = userAccountUpdated.getLastLoginDateTime();
        Assertions.assertNotEquals(lastLoginSecond, lastLoginFirst);
        userPage.assertLastLoginPresent();

        userPage.edit();
        userPage.delete();
        userPage.confirmDelete();

        FailoverUtils.retry(10, () -> {
            long countAfter = userAccountRepository.count();
            Assertions.assertEquals(countBefore-1, countAfter);
            return null;
        });
    }*/

    /*@Test
    public void testBindIdToAccountAndConflict() throws Exception {

        IndexPage indexPage = new IndexPage(urlPrefix);
        indexPage.openPage();

        long countInitial = userAccountRepository.count();
        //Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        UserProfilePage userPage = new UserProfilePage(urlPrefix, driver);
        final String login600 = "generated_user_600";

        LoginModal loginModal600 = new LoginModal(login600, COMMON_PASSWORD);
        loginModal600.openLoginModal();
        loginModal600.login();
        UserAccount userAccount = userAccountRepository.findByUsername(login600).orElseThrow();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertThisIsYou();
        userPage.edit();

        userPage.bindFacebook();
        long countAfter = userAccountRepository.count();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertHasFacebook();

        Assertions.assertEquals(countInitial, countAfter);
        loginModal600.logout();

        // check that binding is preserved
        Selenide.refresh();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertHasFacebook();

        {
            LoginModal loginModalVk = new LoginModal();
            loginModalVk.openLoginModal();
            loginModalVk.loginVkontakte();

            Assertions.assertEquals(vkontakteLogin, UserNav.getLogin());

            loginModalVk.logout();
        }

        loginModal600.openLoginModal();
        loginModal600.login();
        userPage.edit();
        userPage.bindVkontakte();
        $("body").has(Condition.text("Somebody already taken this vkontakte id"));
    }

    @Test
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
