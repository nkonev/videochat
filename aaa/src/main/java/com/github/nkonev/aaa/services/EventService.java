package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.controllers.UserProfileController;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.*;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.security.PrincipalToCheck;
import com.github.nkonev.aaa.security.UserRoleService;
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

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Autowired
    private UserRoleService userRoleService;

    public void notifyProfileCreated(UserAccount userAccount) {
        var data = new UserAccountCreatedEventGroupDTO(
            userAccount.id(),
            "user_account_created",
            userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.knownAdmin(), userAccount),
            UserAccountConverter.convertToUserAccountDTO(userAccount)
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountCreatedEventGroup");
            return message;
        });
    }

    public void notifyProfileUpdated(UserAccount userAccount) {
        var data = new UserAccountEventGroupDTO(
            userAccount.id(),
            "user_account_changed",
            userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(UserAccountConverter.convertToUserAccountDetailsDTO(userAccount), userRoleService), userAccount),
            userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.knownAdmin(), userAccount),
            UserAccountConverter.convertToUserAccountDTO(userAccount)
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountEventGroup");
            return message;
        });
    }

    public void notifyProfileDeleted(long userId) {
        var data = new UserAccountDeletedEventDTO(
            userId,
            "user_account_deleted"
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountDeletedEvent");
            return message;
        });
    }

    public void notifySessionsKilled(long userId, ForceKillSessionsReasonType reasonType) {
        var data = new UserSessionsKilledEventDTO(
            userId,
            "user_sessions_killed",
            reasonType
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserSessionsKilledEvent");
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
