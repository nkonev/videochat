package com.github.nkonev.aaa.dto;

public record SearchUsersRequestDTO(
    int size,
    long startingFromItemId,
    boolean reverse,
    boolean hasHash,
    String searchString
) {
}
