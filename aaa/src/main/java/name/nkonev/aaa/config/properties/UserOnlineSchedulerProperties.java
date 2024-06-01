package name.nkonev.aaa.config.properties;

public record UserOnlineSchedulerProperties(
    boolean enabled,
    int batchSize,
    String cron
) {
}
