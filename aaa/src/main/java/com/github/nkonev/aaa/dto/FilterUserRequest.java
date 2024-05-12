package name.nkonev.aaa.dto;

public record FilterUserRequest(
    String searchString,
    long userId
) {
}
