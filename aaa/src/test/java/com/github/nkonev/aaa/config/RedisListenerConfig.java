package com.github.nkonev.aaa.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.services.RedisEventReceiver;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.listener.ChannelTopic;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;
import org.springframework.data.redis.listener.adapter.MessageListenerAdapter;

import static com.github.nkonev.aaa.services.NotifierService.USER_PROFILE_UPDATE;

@Configuration
public class RedisListenerConfig {
    @Bean
    RedisMessageListenerContainer container(RedisConnectionFactory connectionFactory,
                                            MessageListenerAdapter listenerAdapter) {

        RedisMessageListenerContainer container = new RedisMessageListenerContainer();
        container.setConnectionFactory(connectionFactory);
        container.addMessageListener(listenerAdapter, new ChannelTopic(USER_PROFILE_UPDATE));


        return container;
    }

    @Bean
    MessageListenerAdapter listenerAdapter(RedisEventReceiver receiver) {
        return new MessageListenerAdapter(receiver, "receiveMessage");
    }

    @Bean
    RedisEventReceiver receiver(ObjectMapper objectMapper) {
        return new RedisEventReceiver(objectMapper);
    }
}
