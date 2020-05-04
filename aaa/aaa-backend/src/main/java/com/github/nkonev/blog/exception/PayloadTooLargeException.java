package com.github.nkonev.blog.exception;

public class PayloadTooLargeException extends RuntimeException {
    private static final long serialVersionUID = 2747102235764494803L;

    public PayloadTooLargeException() {
    }

    public PayloadTooLargeException(String message) {
        super(message);
    }
}
