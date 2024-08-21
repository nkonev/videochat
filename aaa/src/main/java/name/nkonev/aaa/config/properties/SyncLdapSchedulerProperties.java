package name.nkonev.aaa.config.properties;

public record SyncLdapSchedulerProperties(
    boolean enabled,
    int batchSize
) {
}
