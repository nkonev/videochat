package com.github.nkonev.blog.exception;

public class UnsupportedMessageTypeException extends RuntimeException{
    private static final long serialVersionUID = -4890563984936375272L;

    public UnsupportedMessageTypeException() {}

    public UnsupportedMessageTypeException(String message) {
        super(message);
    }
}
