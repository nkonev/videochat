package name.nkonev.oauth2emu;

import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.WebApplicationType;
import org.springframework.boot.autoconfigure.ImportAutoConfiguration;
import org.springframework.boot.autoconfigure.web.client.RestTemplateAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.DispatcherServletAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.ServletWebServerFactoryAutoConfiguration;
import org.springframework.boot.autoconfigure.web.servlet.WebMvcAutoConfiguration;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;


import java.net.URI;
import java.util.List;

import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.*;

@Configuration
@ComponentScan
@RestController
@ImportAutoConfiguration({
        ServletWebServerFactoryAutoConfiguration.class,
        DispatcherServletAutoConfiguration.class,
        WebMvcAutoConfiguration.class,
        RestTemplateAutoConfiguration.class,
})
@Import(RestTemplateConfig.class)
public class EmulatorServersController {

    private static final Logger LOGGER = LoggerFactory.getLogger(EmulatorServersController.class);

    @Autowired
    private RestTemplate restTemplate;

    @Value("${custom.template-engine-url-prefix}")
    protected String templateEngineUrlPrefix;

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

    @PutMapping("/recreate-oauth2-mocks")
    public void commandReceiver() {
        LOGGER.info("Removing oauth2-aware users");

        List<String> users = List.of(facebookLogin, vkontakteLogin, googleLogin);

        try {
            restTemplate.put(URI.create(templateEngineUrlPrefix + "/internal/reset"), users);
        } catch (Exception e) {
            LOGGER.warn("Error during resetting aaa: {}", e.getMessage());
        }

        LOGGER.info("Resetting emulators");
        OAuth2EmulatorServers.resetFacebookEmulator();
        OAuth2EmulatorServers.resetVkontakteEmulator();
        OAuth2EmulatorServers.resetGoogleEmulator();

        LOGGER.info("Configuring emulators");
        OAuth2EmulatorServers.configureFacebookEmulator(templateEngineUrlPrefix);
        OAuth2EmulatorServers.configureVkontakteEmulator(templateEngineUrlPrefix);
        OAuth2EmulatorServers.configureGoogleEmulator(templateEngineUrlPrefix);
        LOGGER.info("Emulators were configured");
    }
}
