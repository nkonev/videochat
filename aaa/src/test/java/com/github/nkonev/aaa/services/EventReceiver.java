package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.dto.UserAccountDTO;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Component;

import java.util.concurrent.ConcurrentLinkedQueue;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_PROFILE_EVENTS_NAME;

@Component
public class EventReceiver {
    private static final Logger LOGGER = LoggerFactory.getLogger(EventReceiver.class);

    private final ConcurrentLinkedQueue<UserAccountDTO> queue = new ConcurrentLinkedQueue<>();

    @RabbitListener(queues = QUEUE_PROFILE_EVENTS_NAME)
    public void listen(UserAccountDTO message) {
        LOGGER.info("Received <" + message + ">");
        queue.add(message);
    }

    public void clear() {
        queue.clear();
    }

    public int size() {
        return queue.size();
    }

    public UserAccountDTO getLast() {
        return queue.poll();
    }
}
