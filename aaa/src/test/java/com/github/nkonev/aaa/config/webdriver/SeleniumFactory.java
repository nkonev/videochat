package com.github.nkonev.aaa.config.webdriver;

import com.codeborne.selenide.WebDriverRunner;
import io.github.bonigarcia.wdm.WebDriverManager;
import io.github.bonigarcia.wdm.config.DriverManagerType;
import org.openqa.selenium.Dimension;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.chrome.ChromeDriver;
import org.openqa.selenium.chrome.ChromeOptions;
import org.openqa.selenium.firefox.FirefoxDriver;
import org.openqa.selenium.firefox.FirefoxDriverLogLevel;
import org.openqa.selenium.firefox.FirefoxOptions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.bridge.SLF4JBridgeHandler;
import org.springframework.beans.factory.FactoryBean;
import org.springframework.core.env.Environment;

import java.util.concurrent.TimeUnit;

/**
 * Created by nik on 07.08.16.
 */
public class SeleniumFactory implements FactoryBean<WebDriver> {


    public static final String FIREFOX_DRIVER_VERSION = "0.21.0"; // https://github.com/mozilla/geckodriver/releases
    public static final String CHROME_DRIVER_VERSION = "89.0.4389.23"; // https://sites.google.com/chromium.org/driver/downloads
    public static final String CUSTOM_SELENIUM_ENABLE = "custom.selenium.start";

    private WebDriver driver;

    private Environment environment;

    private static final Logger LOGGER = LoggerFactory.getLogger(SeleniumFactory.class);

    private SeleniumProperties seleniumConfiguration;

    public SeleniumFactory(SeleniumProperties seleniumConfiguration, Environment environment) throws Exception {
        this.seleniumConfiguration = seleniumConfiguration;
        this.environment = environment;
    }

    @Override
    public WebDriver getObject() throws Exception {
        return driver;
    }

    @Override
    public Class<WebDriver> getObjectType() {
        return WebDriver.class;
    }

    @Override
    public boolean isSingleton() {
        return true;
    }

    public void start() {
        if (!seleniumAllowed()) {
            return;
        }

        // http://www.slf4j.org/legacy.html#jul-to-slf4j
        // http://www.slf4j.org/api/org/slf4j/bridge/SLF4JBridgeHandler.html
        // Optionally remove existing handlers attached to j.u.l root logger
        SLF4JBridgeHandler.removeHandlersForRootLogger();  // (since SLF4J 1.6.5)
        // add SLF4JBridgeHandler to j.u.l's root logger, should be done once during
        // the initialization phase of your application
        SLF4JBridgeHandler.install();

        switch(seleniumConfiguration.getBrowser()) {
            case FIREFOX:
            {
                System.setProperty("webdriver.firefox.logfile", "/dev/null");
                // firefox
                WebDriverManager.getInstance(DriverManagerType.FIREFOX).driverVersion(FIREFOX_DRIVER_VERSION).setup(); // download executables if need and set System.properties
                // https://developer.mozilla.org/en-US/Firefox/Headless_mode
                FirefoxOptions firefoxOptions = new FirefoxOptions();
                firefoxOptions.setLogLevel(FirefoxDriverLogLevel.INFO);
                if (seleniumConfiguration.isHeadless()) {
                    firefoxOptions.addArguments("-headless");
                }
                driver = new FirefoxDriver(firefoxOptions);
                com.codeborne.selenide.Configuration.browser = WebDriverRunner.FIREFOX;
            }
            break;
            case CHROME:
            {
                WebDriverManager.getInstance(DriverManagerType.CHROME).driverVersion(CHROME_DRIVER_VERSION).setup(); // download executables if need and set System.properties
                // https://developers.google.com/web/updates/2017/04/headless-chrome
                ChromeOptions chromeOptions = new ChromeOptions();
                if (seleniumConfiguration.isHeadless()) {
                    chromeOptions.addArguments("--headless");
                }
                driver = new ChromeDriver(chromeOptions);
            }
            break;
        }

        driver.manage().window().setSize(new Dimension(seleniumConfiguration.getWindowWidth(), seleniumConfiguration.getWindowHeight()));
        // http://www.seleniumhq.org/docs/04_webdriver_advanced.jsp#explicit-and-implicit-waits
        driver.manage().timeouts().implicitlyWait(seleniumConfiguration.getImplicitlyWaitTimeout(), TimeUnit.SECONDS); // wait for #findElement()

        // configure selenide
        WebDriverRunner.setWebDriver(driver);
        com.codeborne.selenide.Configuration.timeout = seleniumConfiguration.getSelenideConditionTimeout() * 1000;
        com.codeborne.selenide.Configuration.pollingInterval = seleniumConfiguration.getSelenidePollingInterval();
    }

    public void stop() {
        if (!seleniumAllowed()) {
            return;
        }
        if (driver != null) {
            //Step 4- Close Driver
            driver.close();

            //Step 5- Quit Driver
            driver.quit();
        }

    }

    private boolean seleniumAllowed() {
        return environment.getProperty(CUSTOM_SELENIUM_ENABLE, boolean.class, false);
    }
}
