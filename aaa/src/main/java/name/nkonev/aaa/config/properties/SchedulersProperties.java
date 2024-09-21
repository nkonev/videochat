package name.nkonev.aaa.config.properties;

public record SchedulersProperties(
    UserOnlineSchedulerProperties userOnline,
    SyncLdapSchedulerProperties syncLdap,
    SyncKeycloakSchedulerProperties syncKeycloak
) {
}
