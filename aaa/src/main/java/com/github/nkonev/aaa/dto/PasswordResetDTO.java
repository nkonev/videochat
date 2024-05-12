package name.nkonev.aaa.dto;

import name.nkonev.aaa.Constants;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

import java.util.UUID;

public record PasswordResetDTO(
    @NotNull
    UUID passwordResetToken,

    @Size(min = Constants.MIN_PASSWORD_LENGTH, max = Constants.MAX_PASSWORD_LENGTH)
    @NotEmpty
    String newPassword
) {
}
