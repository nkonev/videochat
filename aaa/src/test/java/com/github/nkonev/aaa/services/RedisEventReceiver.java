package com.github.nkonev.aaa.services;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.nkonev.aaa.dto.UserAccountDTO;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.ConcurrentLinkedQueue;

public class RedisEventReceiver {
    private static final Logger LOGGER = LoggerFactory.getLogger(RedisEventReceiver.class);

    private final ConcurrentLinkedQueue<UserAccountDTO> queue = new ConcurrentLinkedQueue<>();

    public RedisEventReceiver(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    private final ObjectMapper objectMapper;

    public void receiveMessage(String message) throws JsonProcessingException {
        LOGGER.info("Received <" + message + ">");
        UserAccountDTO messageObj = objectMapper.readValue(message.strip(), UserAccountDTO.class);
        queue.add(messageObj);
    }

    public UserAccountDTO getLast() {
        return queue.poll();
    }

    public void clear() {
        queue.clear();
    }

    public long size() {
        return queue.size();
    }
}