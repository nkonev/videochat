package com.github.nkonev.aaa.dto;

public record LockDTO (
    long userId,
    boolean lock
) { }
