package com.github.nkonev.aaa.exception;


public class OAuth2IdConflictException extends RuntimeException {
    public OAuth2IdConflictException(String msg) {
        super(msg);
    }
}
