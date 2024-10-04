package name.nkonev.aaa.dto;

public record UserAccountEventChangedDTO(
    Long userId,
    String eventType,
    UserAccountEventDTO user
) { }
