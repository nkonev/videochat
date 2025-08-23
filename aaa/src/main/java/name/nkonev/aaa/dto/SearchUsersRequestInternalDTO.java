package name.nkonev.aaa.dto;

import java.util.List;

public record SearchUsersRequestInternalDTO(
    int page,
    int size,
    List<Long> userIds,
    String searchString,
    boolean including
) {
}
