package com.github.nkonev.aaa.dto;


import javax.validation.constraints.Email;

public record EditUserDTO (
    String login,

    String avatar,

    String password, // password which user desires

    Boolean removeAvatar, // it handles 3 values: true, false, null

    @Email
    String email,

    String avatarBig,

    String shortInfo
) {

    public EditUserDTO(String login, String avatar, String avatarBig, String shortInfo, String password, String email) {
        this(
                login,
                avatar,
                password,
                null,
                email,
                avatarBig,
                shortInfo
        );
    }

    public EditUserDTO withLogin(String newLogin) {
        return new EditUserDTO(
                newLogin,
                avatar,
                password,
                removeAvatar,
                email,
                avatarBig,
                shortInfo
        );
    }

    public EditUserDTO withPassword(String newPassword) {
        return new EditUserDTO(
                login,
                avatar,
                newPassword,
                removeAvatar,
                email,
                avatarBig,
                shortInfo
        );
    }

    public EditUserDTO withEmail(String newEmail) {
        return new EditUserDTO(
                login,
                avatar,
                password,
                removeAvatar,
                newEmail,
                avatarBig,
                shortInfo
        );
    }
}
