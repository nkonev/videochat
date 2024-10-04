package name.nkonev.aaa.dto;

public record UserAccountEventCreatedDTO(
    Long userId,
    String eventType,
    UserAccountEventDTO user
) { }
