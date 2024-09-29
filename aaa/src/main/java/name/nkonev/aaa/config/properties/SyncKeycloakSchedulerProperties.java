package name.nkonev.aaa.config.properties;

public record SyncKeycloakSchedulerProperties(
    boolean enabled,
    boolean syncEmailVerified,
    int batchSize
) {
}
