package com.github.nkonev.aaa.dto;

import java.util.Set;

public record UserAccountEventDTO(
    ForWho forWho,
    Set<UserRole> forWhoRoles,
    Long userId, // nullable
    String eventType,

    UserAccountDTO userAccount
) {

    public enum ForWho {
        FOR_MYSELF,
        FOR_ROLE,
    }
}
