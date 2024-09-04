package name.nkonev.aaa.dto;


import jakarta.validation.constraints.Email;

public record EditUserDTO (
    String login,

    String avatar,

    String password, // password which user desires

    Boolean removeAvatar, // it handles 3 values: true, false, null

    @Email
    String email,

    String avatarBig,

    Boolean removeShortInfo, // it handles 3 values: true, false, null
    String shortInfo,
    String loginColor,
    Boolean removeLoginColor
) {

    public EditUserDTO(String login, String avatar, String avatarBig, String shortInfo, String password, String email) {
        this(
                login,
                avatar,
                password,
                null,
                email,
                avatarBig,
                null,
                shortInfo,
                null,
                null
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
                removeShortInfo,
                shortInfo,
                loginColor,
                removeLoginColor
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
                removeShortInfo,
                shortInfo,
                loginColor,
                removeLoginColor
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
                removeShortInfo,
                shortInfo,
                loginColor,
                removeLoginColor
        );
    }

    public EditUserDTO withAvatar(String newAvatar) {
        return new EditUserDTO(
            login,
            newAvatar,
            password,
            removeAvatar,
            email,
            avatarBig,
            removeShortInfo,
            shortInfo,
            loginColor,
            removeLoginColor
        );
    }

    public EditUserDTO withAvatarBig(String newAvatar) {
        return new EditUserDTO(
            login,
            avatar,
            password,
            removeAvatar,
            email,
            newAvatar,
            removeShortInfo,
            shortInfo,
            loginColor,
            removeLoginColor
        );
    }

    public EditUserDTO withShortInfo(String newShortInfo) {
        return new EditUserDTO(
            login,
            avatar,
            password,
            removeAvatar,
            email,
            avatarBig,
            removeShortInfo,
            newShortInfo,
            loginColor,
            removeLoginColor
        );
    }

    public EditUserDTO withLoginColor(String newLoginColor) {
        return new EditUserDTO(
            login,
            avatar,
            password,
            removeAvatar,
            email,
            avatarBig,
            removeShortInfo,
            shortInfo,
            newLoginColor,
            removeLoginColor
        );
    }

}
