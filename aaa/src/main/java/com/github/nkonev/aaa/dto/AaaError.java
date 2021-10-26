package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonInclude;

import java.util.ArrayList;
import java.util.Collection;

public record AaaError(
        int status,
        String error,
        String message,
        String timeStamp,
        Collection<ValidationError> validationErrors,
        @JsonInclude(JsonInclude.Include.NON_NULL)
        String exception,
        @JsonInclude(JsonInclude.Include.NON_NULL)
        String trace
) {

    public AaaError(
            int status,
            String error,
            String message,
            String timeStamp,
            String exception,
            String trace
    ) {
        this(
                status,
                error,
                message,
                timeStamp,
                new ArrayList<>(),
                exception,
                trace
        );
    }

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
                new ArrayList<>(),
                null,
                null
        );
    }

    public AaaError(
            int status,
            String error,
            String message,
            String timeStamp,
            Collection<ValidationError> validationErrors
    ) {
        this(
                status,
                error,
                message,
                timeStamp,
                validationErrors,
                null,
                null
        );
    }
}
