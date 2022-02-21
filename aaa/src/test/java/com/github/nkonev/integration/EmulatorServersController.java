package com.github.nkonev.integration;

import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.config.WebConfig;
import com.github.nkonev.aaa.it.OAuth2EmulatorTests;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.repository.redis.UserConfirmationTokenRepository;
import io.opentracing.contrib.spring.tracer.configuration.TracerAutoConfiguration;
import org.springframework.boot.WebApplicationType;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.data.redis.repository.configuration.EnableRedisRepositories;
import org.springframework.security.config.annotation.web.builders.WebSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configuration.WebSecurityConfigurerAdapter;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

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

    public static void main(String[] args) {
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            try {
                tearDownClass();
            } catch (Exception e) {
                e.printStackTrace();
            }
        }));

        setUpClass();

        new SpringApplicationBuilder()
                .profiles("integration_test")
                .properties("spring.config.location=classpath:/config/application-integration_test.yml")
                .web(WebApplicationType.SERVLET)
                .sources(
                    EmulatorServersController.class,
                    CustomConfig.class,
                    WebConfig.class
                )
                .run(args);
    }

    @PostMapping("/recreate-oauth2-mocks")
    public void commandReceiver() throws InterruptedException {
        resetFacebookEmulator();
        resetVkontakteEmulator();
        resetGoogleEmulator();

        configureFacebookEmulator();
        configureVkontakteEmulator();
        configureGoogleEmulator();
    }
}
