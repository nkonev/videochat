package com.github.nkonev.aaa.config;

import org.springframework.amqp.core.*;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_ONLINE_EVENTS_NAME;
import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_PROFILE_EVENTS_NAME;

@Configuration
public class RabbitMqTestConfig {

    // see in chat/listener/rabbitmq.go

    public static final String QUEUE_PROFILE_TEST = "aaa-events-test-queue";

    @Bean
    public Exchange aaaExchange() {
        return new FanoutExchange(QUEUE_PROFILE_EVENTS_NAME, true, false);
    }

    @Bean
    public Queue aaaEvents() {
        return new Queue(QUEUE_PROFILE_TEST, true, false, true);
    }

    @Bean
    public Binding aaaEventsBinding() {
        return BindingBuilder.bind(aaaEvents()).to(aaaExchange()).with("").noargs();
    }

    // see in event/listener/rabbitmq.go
    @Bean
    public Exchange asyncEventsFanoutExchange() {
        return new FanoutExchange(QUEUE_ONLINE_EVENTS_NAME, true, false);
    }

}
