package com.github.nkonev.aaa.services;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.dto.UserAccountDTO;
import com.github.nkonev.aaa.dto.UserAccountDeletedEventDTO;
import com.github.nkonev.aaa.dto.UserAccountEventGroupDTO;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.util.concurrent.ConcurrentLinkedQueue;

import static com.github.nkonev.aaa.config.RabbitMqTestConfig.QUEUE_PROFILE_TEST;

@Component
public class EventReceiver {
    private static final Logger LOGGER = LoggerFactory.getLogger(EventReceiver.class);

    @Autowired
    private ObjectMapper objectMapper;

    private final ConcurrentLinkedQueue<UserAccountDTO> changedQueue = new ConcurrentLinkedQueue<>();

    private final ConcurrentLinkedQueue<UserAccountDeletedEventDTO> deletedQueue = new ConcurrentLinkedQueue<>();

    @RabbitListener(queues = QUEUE_PROFILE_TEST)
    public void listen(org.springframework.amqp.core.Message message) throws IOException {
        LOGGER.info("Received <" + message + ">");
        switch (message.getMessageProperties().getType()) {
            case "dto.UserAccountDeletedEvent": {
                var m = objectMapper.readValue(message.getBody(), UserAccountDeletedEventDTO.class);
                deletedQueue.add(m);
                break;
            }
            case "dto.UserAccountEventGroup": {
                var m = objectMapper.readValue(message.getBody(), UserAccountEventGroupDTO.class);
                changedQueue.add(m.forRoleUser());
                break;
            }
            default: {
                LOGGER.warn("Unknown type: {}", message.getMessageProperties().getType());
            }
        }
    }

    public void clearChanged() {
        changedQueue.clear();
    }

    public int sizeChanged() {
        return changedQueue.size();
    }

    public UserAccountDTO getLastChanged() {
        return changedQueue.poll();
    }


    public void clearDeleted() {
        deletedQueue.clear();
    }

    public int sizeDeleted() {
        return deletedQueue.size();
    }

    public UserAccountDeletedEventDTO getLastDeleted() {
        return deletedQueue.poll();
    }
}
