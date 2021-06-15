package com.github.nkonev.aaa.config.webdriver;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Scope;

/**
 * Created by nik on 04.10.16.
 */
@ConfigurationProperties(prefix = "custom.selenium")
public class SeleniumProperties {

    /**
     * in seconds
     */
    private int implicitlyWaitTimeout;

    private int windowWidth;

    private int windowHeight;

    /**
     * in milliseconds
     */
    private int selenidePollingInterval;

    public int getImplicitlyWaitTimeout() {
        return implicitlyWaitTimeout;
    }

    public void setImplicitlyWaitTimeout(int implicitlyWaitTimeout) {
        this.implicitlyWaitTimeout = implicitlyWaitTimeout;
    }

    public int getWindowWidth() {
        return windowWidth;
    }

    public void setWindowWidth(int windowWidth) {
        this.windowWidth = windowWidth;
    }

    public int getWindowHeight() {
        return windowHeight;
    }

    public void setWindowHeight(int windowHeight) {
        this.windowHeight = windowHeight;
    }

    public int getSelenidePollingInterval() {
        return selenidePollingInterval;
    }

    public void setSelenidePollingInterval(int selenidePollingInterval) {
        this.selenidePollingInterval = selenidePollingInterval;
    }
}
