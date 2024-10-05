package name.nkonev.aaa.services;

import com.fasterxml.jackson.databind.ObjectMapper;
import name.nkonev.aaa.dto.UserAccountDTO;
import name.nkonev.aaa.dto.UserAccountEventChangedDTO;
import name.nkonev.aaa.dto.UserAccountEventDTO;
import name.nkonev.aaa.dto.UserAccountEventDeletedDTO;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.util.concurrent.ConcurrentLinkedQueue;

import static name.nkonev.aaa.config.RabbitMqTestConfig.QUEUE_PROFILE_TEST;

@Component
public class EventReceiver {
    private static final Logger LOGGER = LoggerFactory.getLogger(EventReceiver.class);

    @Autowired
    private ObjectMapper objectMapper;

    private final ConcurrentLinkedQueue<UserAccountEventDTO> changedQueue = new ConcurrentLinkedQueue<>();

    private final ConcurrentLinkedQueue<UserAccountEventDeletedDTO> deletedQueue = new ConcurrentLinkedQueue<>();

    @RabbitListener(queues = QUEUE_PROFILE_TEST)
    public void listen(org.springframework.amqp.core.Message message) throws IOException {
        LOGGER.info("Received {}", message);
        switch (message.getMessageProperties().getType()) {
            case "dto.UserAccountEventDeleted": {
                var m = objectMapper.readValue(message.getBody(), UserAccountEventDeletedDTO.class);
                deletedQueue.add(m);
                break;
            }
            case "dto.UserAccountEventChanged": {
                var m = objectMapper.readValue(message.getBody(), UserAccountEventChangedDTO.class);
                changedQueue.add(m.user());
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

    public UserAccountEventDTO getLastChanged() {
        return changedQueue.poll();
    }


    public void clearDeleted() {
        deletedQueue.clear();
    }

    public int sizeDeleted() {
        return deletedQueue.size();
    }

    public UserAccountEventDeletedDTO getLastDeleted() {
        return deletedQueue.poll();
    }
}
