package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record SyncKeycloakSchedulerProperties(
    boolean enabled,
    boolean syncEmailVerified,
    boolean syncRoles,
    int batchSize,
    Duration expiration,
    int maxEventsBeforeCanThrottle
) {
}
