package com.github.nkonev.aaa.dto;

import java.util.List;

public record SearchUsersRequestInternalDto(
    int page,
    int size,
    List<Long> userIds,
    String searchString,
    boolean including
) {
}
