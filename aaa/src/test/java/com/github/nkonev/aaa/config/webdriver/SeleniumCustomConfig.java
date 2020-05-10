package com.github.nkonev.aaa.config.webdriver;

import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Scope;
import org.springframework.core.env.Environment;

@Configuration
@EnableConfigurationProperties(SeleniumProperties.class)
public class SeleniumCustomConfig {
    /**
     * @return
     * @throws Exception
     * @Scope("singleton") is need as part of https://github.com/spring-projects/spring-boot/issues/7454
     */
    @Scope("singleton")
    @Bean(initMethod = "start", destroyMethod = "stop")
    public SeleniumFactory seleniumComponent(SeleniumProperties seleniumConfiguration, Environment environment) throws Exception {
        return new SeleniumFactory(seleniumConfiguration, environment);
    }
}
