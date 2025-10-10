package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

import java.util.Set;

public record UserAccountInternalDTO(
        @JsonUnwrapped
        UserAccountDTO userAccountDTO,
        Set<ExternalPermission> permissions
) {
}
