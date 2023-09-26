package com.github.nkonev.oauth2emu;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.WebApplicationType;
import org.springframework.boot.autoconfigure.ImportAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.JdbcTemplateAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.DispatcherServletAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.ServletWebServerFactoryAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.WebMvcAutoConfiguration;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;

@Configuration
@ComponentScan
@RestController
@ImportAutoConfiguration({
        ServletWebServerFactoryAutoConfiguration.class,
        DataSourceAutoConfiguration.class,
        JdbcTemplateAutoConfiguration.class,
        DispatcherServletAutoConfiguration.class,
        WebMvcAutoConfiguration.class
})
public class EmulatorServersController {

    private static final Logger LOGGER = LoggerFactory.getLogger(EmulatorServersController.class);

    @Autowired
    private UserTestService userTestService;

    @Value("${custom.base-url}")
    protected String urlPrefix;

    public static void main(String[] args) {
        new SpringApplicationBuilder()
                .profiles("integration_test")
                .properties("spring.config.location=classpath:/config/emulator.yml")
                .web(WebApplicationType.SERVLET)
                .sources(EmulatorServersController.class)
                .run(args);
    }

    @PostConstruct
    public void postConstruct() {
        OAuth2EmulatorServers.start();
        commandReceiver();
    }

    @PreDestroy
    public void preDestroy() throws Exception {
        OAuth2EmulatorServers.stop();
    }

    @PostMapping("/recreate-oauth2-mocks")
    public void commandReceiver() {
        LOGGER.info("Removing oauth2-aware users");
        userTestService.clearOauthBindingsInDb();

        LOGGER.info("Resetting emulators");
        OAuth2EmulatorServers.resetFacebookEmulator();
        OAuth2EmulatorServers.resetVkontakteEmulator();
        OAuth2EmulatorServers.resetGoogleEmulator();

        LOGGER.info("Configuring emulators");
        OAuth2EmulatorServers.configureFacebookEmulator(urlPrefix);
        OAuth2EmulatorServers.configureVkontakteEmulator(urlPrefix);
        OAuth2EmulatorServers.configureGoogleEmulator(urlPrefix);
    }
}
