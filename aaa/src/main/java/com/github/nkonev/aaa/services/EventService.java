package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.controllers.UserProfileController;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountEventDTO;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Set;

import static com.github.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_PROFILE_EVENTS_NAME;
import static com.github.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_ONLINE_EVENTS_NAME;

@Service
public class EventService {

    @Autowired
    private RabbitTemplate rabbitTemplate;

    public void notifyProfileUpdated(UserAccount userAccount) {
        var data = Set.of(
            new UserAccountEventDTO(
                UserAccountEventDTO.ForWho.FOR_MYSELF,
                null,
                userAccount.id(),
                "user_account_changed",
                UserAccountConverter.convertToUserAccountDTOExtended(userAccount)
            ),
            new UserAccountEventDTO(
                UserAccountEventDTO.ForWho.FOR_ROLE,
                Set.of(UserRole.ROLE_ADMIN),
                null,
                "user_account_changed",
                UserAccountConverter.convertToUserAccountDTO(userAccount)
            ),
            new UserAccountEventDTO(
                UserAccountEventDTO.ForWho.FOR_ROLE,
                Set.of(UserRole.ROLE_USER),
                null,
                "user_account_changed",
                UserAccountConverter.convertToUserAccountDTO(userAccount)
            )
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("[]dto.UserAccountEvent");
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
