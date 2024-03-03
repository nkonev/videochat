package com.github.nkonev.aaa.dto;

public record UserAccountDeletedEventDTO(
    Long userId,
    String eventType
) { }
