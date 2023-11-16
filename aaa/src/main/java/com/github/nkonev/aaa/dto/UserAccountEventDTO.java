package com.github.nkonev.aaa.dto;


public record UserAccountEventDTO(
    ForWho forWho,
    Long userId, // nullable
    String eventType,

    Object userAccount
) {

    public enum ForWho {
        FOR_MYSELF,
        FOR_ROLE_USER,
        FOR_ROLE_ADMIN,
    }
}
