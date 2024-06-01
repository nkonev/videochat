package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record HttpClientProperties(
    Duration connectTimeout,
    Duration readTimeout
) {
}
