package name.nkonev.aaa.dto;

public record UserAccountDeletedEventDTO(
    long userId,
    String eventType
) { }
