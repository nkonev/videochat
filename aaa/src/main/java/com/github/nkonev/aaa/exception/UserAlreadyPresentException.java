package com.github.nkonev.aaa.exception;

public class UserAlreadyPresentException extends RuntimeException {

    private static final long serialVersionUID = 2885108427978294154L;

    public UserAlreadyPresentException(String message) {
        super(message);
    }

    public UserAlreadyPresentException() { }
}
