package com.github.nkonev.blog.dto;

import javax.validation.constraints.NotEmpty;

public class OwnerDTO {
    private long id;

    @NotEmpty
    private String login;

    private String avatar;

    public OwnerDTO() {
    }

    public OwnerDTO(long id, @NotEmpty String login, String avatar) {
        this.id = id;
        this.login = login;
        this.avatar = avatar;
    }

    public long getId() {
        return id;
    }

    public void setId(long id) {
        this.id = id;
    }

    public String getLogin() {
        return login;
    }

    public void setLogin(String login) {
        this.login = login;
    }

    public String getAvatar() {
        return avatar;
    }

    public void setAvatar(String avatar) {
        this.avatar = avatar;
    }
}
