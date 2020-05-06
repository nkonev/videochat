package com.github.nkonev.blog.dto;

import java.util.ArrayList;
import java.util.Collection;

public class BlogError {
    private int status;
    private String error;
    private String message;
    private String timeStamp;
    private Collection<ValidationError> validationErrors=new ArrayList<>();

    public BlogError(int status, String error, String message, String timeStamp) {
        this.status = status;
        this.error = error;
        this.message = message;
        this.timeStamp = timeStamp;
    }

    public BlogError(int status, String error, String message, String timeStamp, Collection<ValidationError> validationErrors) {
        this(status, error, message, timeStamp);
        this.validationErrors = validationErrors;
    }

    public BlogError() { }

    public int getStatus() {
        return status;
    }

    public String getError() {
        return error;
    }

    public String getMessage() {
        return message;
    }

    public String getTimeStamp() {
        return timeStamp;
    }

    public void setStatus(int status) {
        this.status = status;
    }

    public void setError(String error) {
        this.error = error;
    }

    public void setMessage(String message) {
        this.message = message;
    }

    public void setTimeStamp(String timeStamp) {
        this.timeStamp = timeStamp;
    }

    public Collection<ValidationError> getValidationErrors() {
        return validationErrors;
    }

    public void setValidationErrors(Collection<ValidationError> validationErrors) {
        this.validationErrors = validationErrors;
    }
}
