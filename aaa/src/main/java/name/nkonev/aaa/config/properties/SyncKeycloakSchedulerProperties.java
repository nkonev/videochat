package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record SyncKeycloakSchedulerProperties(
    boolean enabled,
    boolean syncEmailVerified,
    boolean syncRoles,
    String cron,
    int batchSize,
    Duration expiration
) {
}
