package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record UserOnlineSchedulerProperties(
    boolean enabled,
    int batchSize,
    String cron,
    Duration expiration
) {
}
