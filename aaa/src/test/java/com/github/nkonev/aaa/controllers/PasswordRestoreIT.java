/*package com.github.nkonev.aaa.controllers;

import com.codeborne.selenide.Condition;
import com.codeborne.selenide.Selenide;
import com.github.nkonev.blog.CommonTestConstants;
import com.github.nkonev.blog.extensions.GreenMailExtension;
import com.github.nkonev.blog.extensions.GreenMailExtensionFactory;
import com.github.nkonev.blog.integration.AbstractItTestRunner;
import com.github.nkonev.blog.pages.object.LoginModal;
import com.github.nkonev.blog.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import com.sun.mail.imap.IMAPMessage;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import javax.mail.Message;

import static com.codeborne.selenide.Selenide.$;

public class PasswordRestoreIT extends AbstractItTestRunner {

    @RegisterExtension
    protected GreenMailExtension greenMail = GreenMailExtensionFactory.build();


    @Test
    public void restorePassword() throws Exception {
        final String user = "forgot-password-user";
        final String email = "forgot-password-user@example.com";
        final String newPassword = "olololo1234";
        Selenide.open(urlPrefix + "/password-restore");

        $("input#email").shouldBe(CLICKABLE).setValue(email);
        $("button#send").shouldBe(CLICKABLE).click();

        $(".check-your-email").waitUntil(Condition.text("check your email"), 1000 * 10);

        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = r.getMessages(email);
            Assertions.assertEquals(1, messages.length, "backend should sent one email");
            IMAPMessage imapMessage = (IMAPMessage)messages[0];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            // perform confirm
            driver.navigate().to(parsedUrl);
            $(".password-reset-enter-new").should(Condition.text("Please enter new password"));
        }

        $(".password-reset-enter-new input#new-password").waitUntil(Condition.visible, 1000 * 10);
        $(".password-reset-enter-new input#new-password").setValue(newPassword);
        $(".password-reset-enter-new button#set-password").shouldBe(CLICKABLE).click();

        $("body").waitUntil(Condition.text("Now you can login with new password"), 1000 * 10);

        LoginModal loginModal = new LoginModal(user, newPassword);
        loginModal.openLoginModal();
        loginModal.login();
        $("body").shouldHave(Condition.text(CommonTestConstants.NON_DELETABLE_POST_TITLE));
    }

}
*/