package com.github.nkonev.aaa.dto;

public record UserAccountDeletedEventDTO(
    long userId,
    String eventType
) { }
