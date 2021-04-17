package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import static com.github.nkonev.aaa.config.RabbitMqConfig.QUEUE_EVENTS_NAME;

@Service
public class NotifierService {

    @Autowired
    private RabbitTemplate rabbitTemplate;

    public void notifyProfileUpdated(UserAccount userAccount) {
        rabbitTemplate.convertAndSend(QUEUE_EVENTS_NAME, UserAccountConverter.convertToUserAccountDTO(userAccount));
    }
}
