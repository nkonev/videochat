package com.github.nkonev.aaa.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.amqp.core.Queue;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.amqp.support.converter.Jackson2JsonMessageConverter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class RabbitMqConfig {

    public static final String EXCHANGE_PROFILE_EVENTS_NAME = "aaa-profile-events-exchange";

    public static final String EXCHANGE_ONLINE_EVENTS_NAME = "async-events-exchange";

    public static final String QUEUE_USER_CONFIRMATION_EMAILS_NAME = "aaa-internal-user-confirmation-emails";
    public static final String QUEUE_PASSWORD_RESET_EMAILS_NAME = "aaa-internal-password-reset-email";

    public static final String QUEUE_ARBITRARY_EMAILS_NAME = "aaa-internal-arbitrary-email";

    @Autowired
    private ObjectMapper objectMapper;

    @Bean
    public RabbitTemplate rabbitTemplate(final ConnectionFactory connectionFactory) {
        final var rabbitTemplate = new RabbitTemplate(connectionFactory);
        rabbitTemplate.setMessageConverter(producerJackson2MessageConverter());
        return rabbitTemplate;
    }

    @Bean
    public Jackson2JsonMessageConverter producerJackson2MessageConverter() {
        return new Jackson2JsonMessageConverter(objectMapper);
    }

    @Bean
    public Queue userConfirmationTokenRequests() {
        return new Queue(QUEUE_USER_CONFIRMATION_EMAILS_NAME, true);
    }

    @Bean
    public Queue passwordResetTokenRequests() {
        return new Queue(QUEUE_PASSWORD_RESET_EMAILS_NAME, true);
    }

    @Bean
    public Queue arbitraryEmails() {
        return new Queue(QUEUE_ARBITRARY_EMAILS_NAME, true);
    }

}
