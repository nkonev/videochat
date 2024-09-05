package name.nkonev.aaa.dto;

public record UserAccountEventDeletedDTO(
    long userId,
    String eventType
) { }
