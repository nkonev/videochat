package com.github.nkonev.integration;

import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.config.WebConfig;
import com.github.nkonev.aaa.it.OAuth2EmulatorTests;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import io.opentracing.contrib.spring.tracer.configuration.TracerAutoConfiguration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.WebApplicationType;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.data.redis.repository.configuration.EnableRedisRepositories;
import org.springframework.security.config.annotation.web.builders.WebSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configuration.WebSecurityConfigurerAdapter;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;

@Configuration
@EnableWebSecurity
class IgnoreAllSecurityConfiguration extends WebSecurityConfigurerAdapter {
    @Override
    public void configure(WebSecurity web) throws Exception {
        web.ignoring().antMatchers("/**");
    }
}

@SpringBootApplication(exclude = {TracerAutoConfiguration.class})
@RestController
@EnableRedisRepositories(basePackageClasses = {UserConfirmationTokenRepository.class})
@EnableJdbcRepositories(basePackageClasses = {UserAccountRepository.class})
public class EmulatorServersController extends OAuth2EmulatorTests {

    private static final Logger LOGGER = LoggerFactory.getLogger(EmulatorServersController.class);

    public static void main(String[] args) {
        new SpringApplicationBuilder()
                .profiles("integration_test")
                .properties("spring.config.location=classpath:/config/emulator.yml")
                .web(WebApplicationType.SERVLET)
                .sources(
                    EmulatorServersController.class,
                    CustomConfig.class,
                    WebConfig.class
                )
                .run(args);
    }

    @PostConstruct
    public void postConstruct() {
        setUpClass();
    }

    @PreDestroy
    public void preDestroy() throws Exception {
        tearDownClass();
    }

    @PostMapping("/recreate-oauth2-mocks")
    public void commandReceiver() throws InterruptedException {
        LOGGER.info("Resetting emulators");
        resetFacebookEmulator();
        resetVkontakteEmulator();
        resetGoogleEmulator();

        LOGGER.info("Configuring emulators");
        configureFacebookEmulator();
        configureVkontakteEmulator();
        configureGoogleEmulator();
    }
}
