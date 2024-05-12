package name.nkonev.aaa.dto;

public record UserExists(
    long userId,
    boolean exists
) {
}
