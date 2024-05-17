package name.nkonev.aaa.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.time.Duration;

@ConfigurationProperties(prefix = "custom.http-client")
public record HttpClientProperties(
    Duration connectTimeout,
    Duration readTimeout
) { }
