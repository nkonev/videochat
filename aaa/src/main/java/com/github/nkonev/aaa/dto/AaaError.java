package com.github.nkonev.aaa.dto;

import java.util.ArrayList;
import java.util.Collection;

public record AaaError(
        int status,
        String error,
        String message,
        String timeStamp,
        Collection<com.github.nkonev.aaa.dto.ValidationError> validationErrors
) {
    public AaaError(
            int status,
            String error,
            String message,
            String timeStamp
    ) {
        this(
                status,
                error,
                message,
                timeStamp,
                new ArrayList<>()
        );
    }
}