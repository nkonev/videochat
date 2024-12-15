package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record SyncLdapSchedulerProperties(
    boolean enabled,
    boolean syncRoles,
    int batchSize,
    Duration expiration,
    int maxEventsBeforeCanThrottle
) {
}
