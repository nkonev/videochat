package com.github.nkonev.blog.dto;


import javax.validation.constraints.Email;
import javax.validation.constraints.NotEmpty;

public class EditUserDTO {
    @NotEmpty
    private String login;

    private String avatar;

    private String password; // password which user desires

    private Boolean removeAvatar;

    @Email
    private String email;

    public EditUserDTO() { }

    public EditUserDTO(String login, String avatar, String password, String email) {
        this.login = login;
        this.avatar = avatar;
        this.password = password;
        this.email = email;
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

    public String getPassword() {
        return password;
    }

    public void setPassword(String password) {
        this.password = password;
    }

    public String getEmail() {
        return email;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    public Boolean getRemoveAvatar() {
        return removeAvatar;
    }

    public void setRemoveAvatar(Boolean removeAvatar) {
        this.removeAvatar = removeAvatar;
    }
}
