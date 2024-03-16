package com.github.nkonev.aaa.dto;

public record SearchUsersRequestDto(
    int size,
    long startingFromItemId,
    boolean reverse,
    boolean hasHash,
    String searchString
) {
}
