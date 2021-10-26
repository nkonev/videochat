package com.github.nkonev.aaa.dto;


import javax.validation.constraints.Email;

public record EditUserDTO (
    String login,

    String avatar,

    String password, // password which user desires

    Boolean removeAvatar, // it handles 3 values: true, false, null

    @Email
    String email,

    String avatarBig
) {

    public EditUserDTO(String login, String avatar, String avatarBig, String password, String email) {
        this(
                login,
                avatar,
                password,
                null,
                email,
                avatarBig
        );
    }

    public EditUserDTO withLogin(String newLogin) {
        return new EditUserDTO(
                newLogin,
                avatar,
                password,
                removeAvatar,
                email,
                avatarBig
        );
    }

    public EditUserDTO withPassword(String newPassword) {
        return new EditUserDTO(
                login,
                avatar,
                newPassword,
                removeAvatar,
                email,
                avatarBig
        );
    }

    public EditUserDTO withEmail(String newEmail) {
        return new EditUserDTO(
                login,
                avatar,
                password,
                removeAvatar,
                newEmail,
                avatarBig
        );
    }
}
