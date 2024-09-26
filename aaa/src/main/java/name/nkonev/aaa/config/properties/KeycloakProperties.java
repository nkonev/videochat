package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record KeycloakProperties(
    ConflictResolveStrategy resolveConflictsStrategy,
    Duration tokenDelta,
    boolean allowUnbind
) {
}
