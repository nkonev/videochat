package name.nkonev.aaa.dto;

import java.util.Set;

public record AdditionalDataDTO(
        boolean enabled,
        boolean expired,
        boolean locked,
        boolean confirmed,
        Set<UserRole> roles
) {
}
