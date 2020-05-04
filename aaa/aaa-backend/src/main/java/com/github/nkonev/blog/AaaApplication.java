package com.github.nkonev.blog;

import com.github.nkonev.blog.config.ApplicationConfig;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Created by nik on 20.05.17.
 */
@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"com.github.nkonev.blog"}
)
@EnableConfigurationProperties({ApplicationConfig.class})
public class AaaApplication {

    public static void main(String[] args) throws Exception {
        SpringApplication.run(AaaApplication.class, args);
    }
}
