package com.github.nkonev.aaa.config;

import org.springframework.amqp.core.Queue;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_EVENTS_NAME;

@Configuration
public class RabbitMqTestConfig {
    @Bean
    public Queue aaaEvents() {
        return new Queue(QUEUE_EVENTS_NAME, false);
    }
}
