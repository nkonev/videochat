package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record TokenProperties(
    Duration ttl
) {
}
