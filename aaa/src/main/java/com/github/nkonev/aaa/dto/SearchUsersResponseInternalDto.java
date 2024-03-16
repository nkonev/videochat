package com.github.nkonev.aaa.dto;

import java.util.List;

public record SearchUsersResponseInternalDto(
    List<UserAccountDTO> users,
    long count
) {
}
