package com.github.nkonev.aaa.dto;

public record UserAccountEventDTO(
    UserAccountDTO userAccount,
    String eventType
) {
}
