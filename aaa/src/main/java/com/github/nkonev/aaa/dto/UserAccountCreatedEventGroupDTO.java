package com.github.nkonev.aaa.dto;

public record UserAccountCreatedEventGroupDTO(
    Long userId,
    String eventType,
    UserAccountDTOExtended forRoleAdmin,
    UserAccountDTO forRoleUser
) { }
