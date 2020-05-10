package com.github.nkonev.aaa;

import com.codeborne.selenide.Condition;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.it.OAuth2EmulatorTests;
import org.junit.jupiter.api.BeforeEach;
import org.openqa.selenium.WebDriver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.test.context.TestPropertySource;

import static com.codeborne.selenide.Selenide.clearBrowserCookies;

@TestPropertySource(properties = {"custom.selenium.enable=true"})
public class AbstractSeleniumRunner extends OAuth2EmulatorTests {

    private Logger LOGGER = LoggerFactory.getLogger(AbstractSeleniumRunner.class);

    // http://www.seleniumhq.org/docs/04_webdriver_advanced.jsp#expected-conditions
    // clickable https://seleniumhq.github.io/selenium/docs/api/java/org/openqa/selenium/support/ui/ExpectedConditions.html#elementToBeClickable-org.openqa.selenium.By-
    public static final Condition[] CLICKABLE = {Condition.exist, Condition.enabled, Condition.visible};

    @BeforeEach
    public void before() {
        LOGGER.debug("Executing before");
        clearBrowserCookies();
    }

    @Autowired
    protected WebDriver driver;

    @Autowired
    protected TestRestTemplate testRestTemplate;

    @Autowired
    protected ObjectMapper objectMapper;


}
