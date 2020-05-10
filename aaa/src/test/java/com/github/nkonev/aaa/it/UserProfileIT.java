/*
package com.github.nkonev.aaa.controllers;

import com.codeborne.selenide.Condition;
import com.codeborne.selenide.Selenide;
import com.github.nkonev.blog.CommonTestConstants;
import com.github.nkonev.blog.FailoverUtils;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.integration.AbstractItTestRunner;
import com.github.nkonev.blog.pages.object.*;
import com.github.nkonev.blog.util.FileUtils;
import com.github.nkonev.blog.webdriver.IntegrationTestConstants;
import com.github.nkonev.blog.webdriver.configuration.SeleniumConfiguration;
import com.github.nkonev.blog.webdriver.selenium.Browser;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebDriver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.namedparam.EmptySqlParameterSource;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;

import static com.codeborne.selenide.Condition.text;
import static com.codeborne.selenide.Selenide.$;
import static com.github.nkonev.blog.CommonTestConstants.COMMON_PASSWORD;
import static com.github.nkonev.blog.pages.object.Buttons.FB;
import static com.github.nkonev.blog.pages.object.Buttons.VK;


public class UserProfileIT extends AbstractItTestRunner {

    @Value(IntegrationTestConstants.USER_ID)
    private int userId;

    @Autowired
    private SeleniumConfiguration seleniumConfiguration;

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    public static class UserProfilePage {
        private static final String BODY = "body";

        private static final Logger LOGGER = LoggerFactory.getLogger(UserProfilePage.class);
        public static final String USER_PROFILE_VIEW_INFO_DATA = ".user-profile-view-info-data";
        private String urlPrefix;
        private WebDriver driver;
        private static final int USER_PROFILE_WAIT = 20000;
        public UserProfilePage(String urlPrefix, WebDriver driver) {
            this.urlPrefix = urlPrefix;
            this.driver = driver;
        }
        public void openPage(int userId) {
            Selenide.open(urlPrefix+ IntegrationTestConstants.Pages.USER+"/"+userId);
        }

        public void sendEnd() {
            $(BODY).sendKeys(Keys.END);
        }


        public void setAvatarImage(String absoluteFilePath) {
            Croppa.setImage(absoluteFilePath);
        }

        // Press Pencil
        public void edit() {
            FailoverUtils.retry(2, () -> {
                $(".user-profile .manage-buttons img.edit-container-pen").click();
                $(".user-profile").waitUntil(Condition.text("Editing profile"), USER_PROFILE_WAIT);
                return null;
            });
        }

        public void assertThisIsYou() {
            $(".user-profile .user-profile-view-info-me .me").waitUntil(Condition.text("This is you"), USER_PROFILE_WAIT);
        }

        public String getAvatarUrl() {
            return $(".user-profile .avatar").getAttribute("src");
        }

        public void setLogin(String login) {
            $(".profile-edit-info input#login").setValue(login);
        }
        public void save(){
            $(".profile-edit button.save").click();
        }
        public void assertLogin(String expected){
            $(".user-profile .user-profile-view-info-data .login").shouldHave(text((expected)));
        }
        public void assertMsg(long expectedId){
            $(".user-profile-msg").shouldHave(Condition.text("Viewing profile #" + expectedId));
        }

        public void setEmail(String email) {
            $(".profile-edit-info input#e-mail").setValue(email);
        }
        public void assertEmail(String expected){
            $(".user-profile .user-profile-view-info-data .email").shouldHave(text((expected)));
        }

        public void removeImage() {
            Croppa.rmImage();
        }

        public void assertLastLoginPresent() {
            $(".user-profile .user-profile-view-info-data .last-login").shouldHave(Condition.text("20"));
        }

        public void delete() {
            $(".profile-edit button.delete").click();
        }

        public void confirmDelete() {
            Dialog.waitForDialog();
            Dialog.clickYes();
        }

        public static final String PROFILE_EDIT_FORM_BINDING = ".profile-edit-info-form-binding .bind";
        public static final String PROFILE_EDIT_FORM_UNBINDING = ".profile-edit-info-form-binding .unbind";

        public void bindFacebook() {
            $(PROFILE_EDIT_FORM_BINDING).find(FB).click();
        }
        public void bindVkontakte() {
            $(PROFILE_EDIT_FORM_BINDING).find(VK).click();
        }

        public void assertHasFacebook() {
            $(USER_PROFILE_VIEW_INFO_DATA).find("a.oauth-fb").shouldHave(Condition.exist);
        }
        public void assertNotHasFacebook() {
            $(USER_PROFILE_VIEW_INFO_DATA).find("a.oauth-fb").shouldNot(Condition.exist);
        }

        public void assertHasVkontakte() {
            $(USER_PROFILE_VIEW_INFO_DATA).find("a.oauth-vk").shouldHave(Condition.exist);
        }

        public void assertNotHasVkontakte() {
            $(USER_PROFILE_VIEW_INFO_DATA).find("a.oauth-vk").shouldNot(Condition.exist);
        }

        public void unBindFacebook() {
            $(PROFILE_EDIT_FORM_UNBINDING).find(FB).click();
        }

        public void unBindVkontakte() {
            $(PROFILE_EDIT_FORM_UNBINDING).find(VK).click();
        }

    }

    private long getUserAvatarCount() {
        return namedParameterJdbcTemplate.queryForObject("select count(*) from images.user_avatar_image;" , EmptySqlParameterSource.INSTANCE, Long.class);
    }


    @Test
    public void userEdit() throws Exception {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        UserProfilePage userPage = new UserProfilePage(urlPrefix, driver);
        final String login = "generated_user_500";
        final long userId = userAccountRepository.findByUsername(login).orElseThrow(()-> new RuntimeException("user Not found")).getId();
        userPage.openPage((int)userId);

        LoginModal loginModal = new LoginModal(login, COMMON_PASSWORD);
        loginModal.openLoginModal();
        loginModal.login();

        userPage.assertThisIsYou();

        String urlOnPageBefore = userPage.getAvatarUrl();
        String urlInNavbarBefore = UserNav.getAvatarUrl();
        Assertions.assertEquals(urlOnPageBefore, urlInNavbarBefore);

        userPage.edit();

        long countBefore = getUserAvatarCount();

        userPage.removeImage();
        userPage.setAvatarImage(FileUtils.getExistsFile("../"+ CommonTestConstants.TEST_IMAGE, CommonTestConstants.TEST_IMAGE).getCanonicalPath());

        final String renamed = "generated_user_500_edit";
        userPage.setLogin(renamed);
        userPage.save();

        userPage.assertThisIsYou();
        userPage.assertLogin(renamed);

        long countAfter = getUserAvatarCount();
        Assertions.assertEquals(countBefore+1, countAfter);

        String urlOnPageAfter = userPage.getAvatarUrl();
        String urlInNavbarAfter = UserNav.getAvatarUrl();
        Assertions.assertFalse(StringUtils.isEmpty(urlOnPageAfter));
        Assertions.assertNotEquals(urlOnPageBefore, urlOnPageAfter);
        Assertions.assertEquals(urlOnPageAfter, urlInNavbarAfter);
    }


    @Test
    public void testUserProfileCorrectlyUpdatedWhenUrlSwitched() {
        UserListIT.UsersPage userListPage = new UserListIT.UsersPage(urlPrefix);
        userListPage.openPage();

        LoginModal loginModal = new LoginModal(user, password);
        loginModal.openLoginModal();
        loginModal.login();

        final String anotherUserLogin = "generated_user_0";
        long anotherUserId = userAccountRepository.findByUsername(anotherUserLogin).orElseThrow(()->new RuntimeException("User Not found")).getId();

        UserProfilePage userProfilePage = new UserProfilePage(null, driver);

        FailoverUtils.retry(2, () -> {
            $(UserListIT.UsersPage.USERS_CONTAINER_SELECTOR)
                    .shouldHave(Condition.text(anotherUserLogin))
                    .findElement(By.linkText(anotherUserLogin))
                    .click();

            userProfilePage.assertMsg(anotherUserId);
            return null;
        });

        userProfilePage.assertLogin(anotherUserLogin);
        String anotherUserAvatarUrl = userProfilePage.getAvatarUrl();

        FailoverUtils.retry(2, () -> {
            UserNav.open();
            UserNav.profile();
            return null;
        });

        final String myLogin = user;
        final long myUserId = userAccountRepository.findByUsername(myLogin).orElseThrow(()->new RuntimeException("User Not found")).getId();

        userProfilePage.assertMsg(myUserId);
        userProfilePage.assertLogin(myLogin);
        String myAvatarUrl = userProfilePage.getAvatarUrl();

        Assertions.assertNotEquals(anotherUserAvatarUrl, myAvatarUrl);
    }

    @Test
    public void testFacebookLogin() throws InterruptedException {
        Assumptions.assumeTrue(Browser.CHROME.equals(seleniumConfiguration.getBrowser()), "Browser must be chrome");

        IndexPage indexPage = new IndexPage(urlPrefix);
        indexPage.openPage();

        LoginModal loginModal = new LoginModal();
        loginModal.openLoginModal();
        loginModal.loginFacebook();

        Assertions.assertTrue(UserNav.getAvatarUrl().endsWith(".png"));
        Assertions.assertEquals(facebookLogin, UserNav.getLogin());


        // now we attempt to change email
        UserProfilePage userPage = new UserProfilePage(urlPrefix, driver);
        UserAccount userAccount = userAccountRepository.findByUsername(facebookLogin).orElseThrow();
        userPage.openPage(userAccount.getId().intValue());
        userPage.assertThisIsYou();
        userPage.edit();
        userPage.setEmail("new-email-for-facebook-user@gmail.not");
        userPage.save();
        userPage.assertEmail("new-email-for-facebook-user@gmail.not");
    }

    @Test
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
    }

    @Test
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

    }
}
*/