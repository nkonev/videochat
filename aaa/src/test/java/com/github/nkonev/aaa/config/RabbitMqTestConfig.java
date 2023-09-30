package com.github.nkonev.aaa.config;

import org.springframework.amqp.core.Exchange;
import org.springframework.amqp.core.FanoutExchange;
import org.springframework.amqp.core.Queue;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_ONLINE_EVENTS_NAME;
import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_PROFILE_EVENTS_NAME;

@Configuration
public class RabbitMqTestConfig {

    // see in chat/listener/rabbitmq.go
    @Bean
    public Queue aaaEvents() {
        return new Queue(QUEUE_PROFILE_EVENTS_NAME, true);
    }

    // see in event/listener/rabbitmq.go
    @Bean
    public Exchange asyncEventsFanoutExchange() {
        return new FanoutExchange(QUEUE_ONLINE_EVENTS_NAME, true, false);
    }

}
