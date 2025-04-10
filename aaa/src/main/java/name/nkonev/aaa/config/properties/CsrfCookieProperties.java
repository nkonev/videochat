package name.nkonev.aaa.config.properties;

import java.time.Duration;

public record CsrfCookieProperties(
    boolean secure,
    String sameSite,
    boolean httpOnly,
    String name, // cookie name
    Duration maxAge
) {
}
