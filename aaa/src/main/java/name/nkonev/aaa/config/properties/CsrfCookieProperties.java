package name.nkonev.aaa.config.properties;

public record CsrfCookieProperties(
    boolean secure,
    String sameSite,
    boolean httpOnly,
    String name // cookie name
) {
}
