package com.github.nkonev.aaa.dto;

public record UserExists(
    long userId,
    boolean exists
) {
}
