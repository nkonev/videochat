package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record SchedulersProperties(
    UserOnlineSchedulerProperties userOnline,
    SyncLdapSchedulerProperties syncLdap,
    SyncKeycloakSchedulerProperties syncKeycloak,
    Duration awaitForTermination,
    int poolSize
) {
}
