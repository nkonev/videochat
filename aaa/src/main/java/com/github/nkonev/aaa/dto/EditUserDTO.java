package com.github.nkonev.aaa.dto;


import javax.validation.constraints.Email;
import javax.validation.constraints.NotEmpty;

public class EditUserDTO {
    private String login;

    private String avatar;

    private String password; // password which user desires

    private Boolean removeAvatar;

    @Email
    private String email;
    private String avatarBig;

    public EditUserDTO() { }

    public EditUserDTO(String login, String avatar, String avatarBig, String password, String email) {
        this.login = login;
        this.avatar = avatar;
        this.avatarBig = avatarBig;
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

    public String getAvatarBig() {
        return avatarBig;
    }

    public void setAvatarBig(String avatarBig) {
        this.avatarBig = avatarBig;
    }
}
