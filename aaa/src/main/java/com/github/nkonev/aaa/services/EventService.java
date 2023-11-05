package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.controllers.UserProfileController;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;

import static com.github.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_PROFILE_EVENTS_NAME;
import static com.github.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_ONLINE_EVENTS_NAME;

@Service
public class EventService {

    @Autowired
    private RabbitTemplate rabbitTemplate;

    public void notifyProfileUpdated(UserAccount userAccount) {
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", UserAccountConverter.convertToUserAccountDTO(userAccount), message -> {
            message.getMessageProperties().setType("dto.UserAccount");
            return message;
        });
    }

    public void notifyOnlineChanged(List<UserProfileController.UserOnlineResponse> userOnline) {
        rabbitTemplate.convertAndSend(EXCHANGE_ONLINE_EVENTS_NAME, "", userOnline, message -> {
            message.getMessageProperties().setType("[]dto.UserOnline");
            return message;
        });
    }
}
