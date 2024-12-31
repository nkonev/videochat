package name.nkonev.aaa.dto;

import name.nkonev.aaa.Constants;
import jakarta.validation.constraints.NotEmpty;
import jakarta.validation.constraints.Size;

public record SetPasswordDTO(
        @Size(min = Constants.MIN_PASSWORD_LENGTH, max = Constants.MAX_PASSWORD_LENGTH)
        @NotEmpty
        String password
) {
}
