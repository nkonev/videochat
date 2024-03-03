package com.github.nkonev.aaa.dto;

public record UserSessionsKilledEventDTO(
    long userId,
    String eventType,
    ForceKillSessionsReasonType reasonType
) { }
