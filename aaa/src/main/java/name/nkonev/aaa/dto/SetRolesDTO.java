package name.nkonev.aaa.dto;

import java.util.Set;

public record SetRolesDTO(
    long userId,
    Set<UserRole> roles
) {
}
