package name.nkonev.aaa.dto;

import java.util.List;

public record SearchUsersResponseInternalDTO(
    List<UserAccountDTO> users,
    long count
) {
}
