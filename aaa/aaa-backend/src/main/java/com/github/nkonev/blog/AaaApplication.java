package com.github.nkonev.blog;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.sleuth.instrument.messaging.TraceMessagingAutoConfiguration;
import org.springframework.cloud.sleuth.instrument.messaging.TraceSpringIntegrationAutoConfiguration;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Created by nik on 04.05.20.
 */
@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"com.github.nkonev.blog"},
        exclude = {TraceMessagingAutoConfiguration.class, TraceSpringIntegrationAutoConfiguration.class}
)
public class AaaApplication {

    public static void main(String[] args) throws Exception {
        SpringApplication.run(AaaApplication.class, args);
    }
}
