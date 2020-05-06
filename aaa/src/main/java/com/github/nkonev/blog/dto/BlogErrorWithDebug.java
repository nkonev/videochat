package com.github.nkonev.blog.dto;

public class BlogErrorWithDebug extends BlogError {
    private String exception;
    private String trace;

    public BlogErrorWithDebug() { }

    public BlogErrorWithDebug(int status, String error, String message, String timeStamp, String exception, String trace) {
        super(status, error, message, timeStamp);
        this.exception = exception;
        this.trace = trace;
    }

    public String getException() {
        return exception;
    }

    public void setException(String exception) {
        this.exception = exception;
    }

    public String getTrace() {
        return trace;
    }

    public void setTrace(String trace) {
        this.trace = trace;
    }
}
