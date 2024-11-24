package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.repository.spring.jdbc.UserListViewRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.services.EventService;
import name.nkonev.aaa.services.LockService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

@Service
public class UserOnlineTask {

    private final UserListViewRepository userListViewRepository;

    private final AaaUserDetailsService aaaUserDetailsService;

    private final EventService eventService;

    private final AaaProperties aaaProperties;

    private final LockService lockService;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserOnlineTask.class);

    private static final String LOCK_NAME = "userOnlineTask";

    public UserOnlineTask(UserListViewRepository userListViewRepository, AaaUserDetailsService aaaUserDetailsService, EventService eventService, AaaProperties aaaProperties, LockService lockService) {
        this.userListViewRepository = userListViewRepository;
        this.aaaUserDetailsService = aaaUserDetailsService;
        this.eventService = eventService;
        this.aaaProperties = aaaProperties;
        this.lockService = lockService;
    }

    @Scheduled(cron = "${custom.schedulers.user-online.cron}")
    public void scheduledTask() {
        if (!aaaProperties.schedulers().userOnline().enabled()) {
            return;
        }

        try (var l = lockService.lock(LOCK_NAME, aaaProperties.schedulers().userOnline().expiration())) {
            if (l.isWasSet()) {
                this.doWork();
            }
        }
    }

    public void doWork() {
        final int pageSize = aaaProperties.schedulers().userOnline().batchSize();
        LOGGER.debug("User online task start, batchSize={}", pageSize);

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
