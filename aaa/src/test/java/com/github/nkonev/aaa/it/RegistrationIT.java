/*package com.github.nkonev.aaa.controllers;

import com.codeborne.selenide.Condition;
import com.github.nkonev.blog.CommonTestConstants;
import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.extensions.GreenMailExtension;
import com.github.nkonev.blog.extensions.GreenMailExtensionFactory;
import com.github.nkonev.blog.integration.AbstractItTestRunner;
import com.github.nkonev.blog.pages.object.LoginModal;
import com.github.nkonev.blog.repository.redis.UserConfirmationTokenRepository;
import com.github.nkonev.blog.util.UrlParser;
import com.icegreen.greenmail.util.Retriever;
import com.sun.mail.imap.IMAPMessage;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.util.UriComponentsBuilder;

import javax.mail.Message;
import java.net.URI;

import static com.codeborne.selenide.Selectors.byText;
import static com.codeborne.selenide.Selenide.$;

public class RegistrationIT extends AbstractItTestRunner {

    @Autowired
    private UserConfirmationTokenRepository userConfirmationTokenRepository;

    private static final Logger LOGGER = LoggerFactory.getLogger(RegistrationIT.class);

    @RegisterExtension
    protected GreenMailExtension greenMail = GreenMailExtensionFactory.build();

    @Test
    public void testConfirmationSuccess() throws Exception {
        // TimeUnit.SECONDS.sleep(10);
        final String email = "newbiewd@example.com";
        final String username = "newbiewd";
        final String password = "password";

        // register
        driver.navigate().to(urlPrefix+"/registration");

        $(byText("Registration")).shouldBe(CLICKABLE).click();
        $(".registration input#login").shouldBe(CLICKABLE).setValue(username);
        $(".registration input#email").shouldBe(CLICKABLE).setValue(email);
        $(".registration input#password").shouldBe(CLICKABLE).setValue(password);
        // Configuration.timeout = 7 * 1000;
        $("button#submit").shouldBe(CLICKABLE).click();

        // we must wait here before invoke greenMail.getImap()
        $(".registration").should(Condition.text("Your confirmation email successfully sent"));

        // Resend
        $(".registration a.registration-confirmation-resend")
                .waitUntil(Condition.visible, 10 * 1000)
                .waitUntil(Condition.enabled, 10 * 1000)
                .click();
        $(".resend-registration-confirmation-token input").shouldBe(Condition.visible).shouldHave(Condition.value(email));
        $(".resend-registration-confirmation-token button").shouldBe(CLICKABLE).click();
        $(".resend-registration-confirmation-token span.email-successfully-sent").shouldBe(Condition.visible);

        // confirm
        try (Retriever r = new Retriever(greenMail.getImap())) {
            Message[] messages = r.getMessages(email);
            Assertions.assertEquals(2, messages.length, "backend should sent one email during registration and I request resend");
            IMAPMessage imapMessage = (IMAPMessage)messages[1];
            String content = (String) imapMessage.getContent();

            String parsedUrl = UrlParser.parseUrlFromMessage(content);

            String tokenUuidString = UriComponentsBuilder.fromUri(new URI(parsedUrl)).build().getQueryParams().get(Constants.Urls.UUID).get(0);
            Assertions.assertTrue(userConfirmationTokenRepository.existsById(tokenUuidString));

            // perform confirm
            driver.navigate().to(parsedUrl);
            Assertions.assertFalse(userConfirmationTokenRepository.existsById(tokenUuidString));
            $(".successfully-confirmed").should(Condition.text("You successfully confirmed, now you can"));
        }

        // login confirmed ok
        $(".successfully-confirmed a").shouldBe(CLICKABLE).click(); // it open login modal
        LoginModal loginModal = new LoginModal(username, password);
        loginModal.login();
        $("body").shouldHave(Condition.text(CommonTestConstants.NON_DELETABLE_POST_TITLE));
    }


}
*/