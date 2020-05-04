package com.github.nkonev.blog.exception;

public class PasswordResetTokenNotFoundException extends RuntimeException {
    private static final long serialVersionUID = 952486328852702273L;

    public PasswordResetTokenNotFoundException(String message) {
        super(message);
    }

    public PasswordResetTokenNotFoundException() {  }
}
