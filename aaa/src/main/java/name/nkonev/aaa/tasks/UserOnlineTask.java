package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.repository.spring.jdbc.UserListViewRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.services.EventService;
import net.javacrumbs.shedlock.spring.annotation.SchedulerLock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

@ConditionalOnProperty("custom.schedulers.user-online.enabled")
@Service
public class UserOnlineTask {

    @Autowired
    private UserListViewRepository userListViewRepository;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private EventService eventService;

    @Autowired
    private AaaProperties aaaProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserOnlineTask.class);

    @Scheduled(cron = "${custom.schedulers.user-online.cron}")
    @SchedulerLock(name = "userOnlineTask")
    public void scheduledTask() {
        final int pageSize = aaaProperties.schedulers().userOnline().batchSize();
        LOGGER.debug("User online task start, userOnlineBatchSize={}", pageSize);

        var shouldContinue = true;
        for (int i = 0; shouldContinue; i++) {
            var chunk = userListViewRepository.findPage(pageSize, i * pageSize);
            shouldContinue = chunk.size() == pageSize;
            var usersOnline = aaaUserDetailsService.getUsersOnlineByUsers(chunk);
            eventService.notifyOnlineChanged(usersOnline);
        }
        LOGGER.debug("User online task finish");
    }
}
