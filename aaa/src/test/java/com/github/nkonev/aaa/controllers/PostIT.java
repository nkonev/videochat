/*package com.github.nkonev.aaa.controllers;

import com.codeborne.selenide.Condition;
import com.github.nkonev.blog.pages.object.*;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URI;

import static com.codeborne.selenide.Selenide.$;
import static com.codeborne.selenide.Selenide.open;


public class PostIT extends OAuth2EmulatorTests {

    private static final Logger LOGGER = LoggerFactory.getLogger(PostIT.class);

    private static final long POST_WITHOUT_COMMENTS = 90;
    private static final long POST_FOR_EDIT_COMMENTS = 80;

    public static class PostViewPage {
        private static final String POST_PART = "/post/";
        private String urlPrefix;

        public PostViewPage(String urlPrefix) {
            this.urlPrefix = urlPrefix;
        }

        public void openPost(long id) {
            open(getUrl(id));
        }

        public String getUrl(long id) {
            return urlPrefix+POST_PART+id;
        }

        public void assertTitle(String expected) {
            $(".post .post-title").waitUntil(Condition.visible, 20 * 1000).should(Condition.text(expected));
        }

        public void assertText(String expected) {
            $(".post .post-content").waitUntil(Condition.visible, 20 * 1000).should(Condition.text(expected));
        }

        public void edit() {
            $(".post-head .edit-container-pen").shouldBe(CLICKABLE).click();
        }

        public void delete() {
            $(".post-head .remove-container-x")
                    .waitUntil(Condition.visible, 20 * 1000)
                    .waitUntil(Condition.enabled, 20 * 1000)
                    .shouldBe(CLICKABLE).click();
        }

        public void confirmDelete() {
            Dialog.waitForDialog();
            Dialog.clickYes();
        }
    }

    @Test
    public void facebookLoginFromPostPageReturnsToPostPage() throws Exception {
        PostViewPage postViewPage = new PostViewPage(urlPrefix);
        postViewPage.openPost(POST_FOR_EDIT_COMMENTS);

        LoginModal loginModal = new LoginModal();
        loginModal.openLoginModal();
        loginModal.loginFacebook();

        Assertions.assertEquals(facebookLogin, UserNav.getLogin());

        long postId = getPostId(URI.create(driver.getCurrentUrl()).toURL());
        Assertions.assertEquals(POST_FOR_EDIT_COMMENTS, postId);
    }

}
*/