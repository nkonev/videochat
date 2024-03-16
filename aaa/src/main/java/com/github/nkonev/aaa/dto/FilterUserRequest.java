package com.github.nkonev.aaa.dto;

public record FilterUserRequest(
    String searchString,
    long userId
) {
}
