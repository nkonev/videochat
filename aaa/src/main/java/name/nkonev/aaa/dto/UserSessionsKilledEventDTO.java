package name.nkonev.aaa.dto;

public record UserSessionsKilledEventDTO(
    long userId,
    String eventType,
    ForceKillSessionsReasonType reasonType
) { }
