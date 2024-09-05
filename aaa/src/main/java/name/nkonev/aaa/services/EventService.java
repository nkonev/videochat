package name.nkonev.aaa.services;

import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.*;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;

import static name.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_PROFILE_EVENTS_NAME;
import static name.nkonev.aaa.config.RabbitMqConfig.EXCHANGE_ONLINE_EVENTS_NAME;

@Service
public class EventService {

    @Autowired
    private RabbitTemplate rabbitTemplate;

    public void notifyProfileCreated(UserAccount userAccount) {
        var data = new UserAccountEventCreatedDTO(
            userAccount.id(),
            "user_account_created",
            UserAccountConverter.convertToUserAccountDTO(userAccount)
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountEventCreated");
            return message;
        });
    }

    public void notifyProfileUpdated(UserAccount userAccount) {
        var data = new UserAccountEventChangedDTO(
            userAccount.id(),
            "user_account_changed",
            UserAccountConverter.convertToUserAccountDTO(userAccount)
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountEventChanged");
            return message;
        });
    }

    public void notifyProfileDeleted(long userId) {
        var data = new UserAccountEventDeletedDTO(
            userId,
            "user_account_deleted"
        );
        rabbitTemplate.convertAndSend(EXCHANGE_PROFILE_EVENTS_NAME, "", data, message -> {
            message.getMessageProperties().setType("dto.UserAccountEventDeleted");
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

    public void notifyOnlineChanged(List<UserOnlineResponse> userOnline) {
        rabbitTemplate.convertAndSend(EXCHANGE_ONLINE_EVENTS_NAME, "", userOnline, message -> {
            message.getMessageProperties().setType("[]dto.UserOnline");
            return message;
        });
    }
}
