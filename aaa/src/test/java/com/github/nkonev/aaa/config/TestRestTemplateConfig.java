package com.github.nkonev.aaa.config;

import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.web.client.RestTemplateBuilder;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.JdkClientHttpRequestFactory;

import java.time.Duration;
import java.time.temporal.ChronoUnit;

@Configuration
public class TestRestTemplateConfig {

    @Bean
    public TestRestTemplate testRestTemplate() {
        RestTemplateBuilder builder = new RestTemplateBuilder();
        builder = builder
            .setConnectTimeout(Duration.of(10, ChronoUnit.SECONDS))
            .setReadTimeout(Duration.of(20, ChronoUnit.SECONDS))
            .requestFactory(JdkClientHttpRequestFactory.class);

        return new TestRestTemplate(builder);
    }
}
