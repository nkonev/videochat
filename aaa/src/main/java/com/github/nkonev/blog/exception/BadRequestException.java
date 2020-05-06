package com.github.nkonev.blog.exception;

public class BadRequestException extends RuntimeException {
    private static final long serialVersionUID = -1966525267992815690L;

    public BadRequestException() { }

    public BadRequestException(String message) {
        super(message);
    }
}
