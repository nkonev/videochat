package com.github.nkonev.aaa;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Created by nik on 04.05.20.
 */
@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"com.github.nkonev.aaa"}
)
public class AaaApplication {

    public static void main(String[] args) throws Exception {
        SpringApplication.run(AaaApplication.class, args);
    }
}
