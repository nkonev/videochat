package com.github.nkonev.aaa.dto;

import java.util.ArrayList;
import java.util.Collection;

public record AaaErrorWithDebug(
        int status,
        String error,
        String message,
        String timeStamp,
        Collection<ValidationError> validationErrors,
        String exception,
        String trace
) {

    public AaaErrorWithDebug(
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
}
