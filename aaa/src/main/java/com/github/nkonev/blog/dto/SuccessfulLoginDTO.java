package com.github.nkonev.blog.dto;

public class SuccessfulLoginDTO {
    private long id;

    private String message;

    public SuccessfulLoginDTO() { }

    public SuccessfulLoginDTO(long id, String message) {
        this.id = id;
        this.message = message;
    }

    public long getId() {
        return id;
    }

    public void setId(long id) {
        this.id = id;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }
}
