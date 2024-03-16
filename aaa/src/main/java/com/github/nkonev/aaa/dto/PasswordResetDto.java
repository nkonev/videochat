package com.github.nkonev.aaa.dto;

import com.github.nkonev.aaa.Constants;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

import java.util.UUID;

public record PasswordResetDto(
    @NotNull
    UUID passwordResetToken,

    @Size(min = Constants.MIN_PASSWORD_LENGTH, max = Constants.MAX_PASSWORD_LENGTH)
    @NotEmpty
    String newPassword
) {
}
