package name.nkonev.aaa.dto;

import java.util.List;

public record SearchUsersResponseDTO(
        List<UserAccountDTOExtended> items,
        boolean hasNext
) {
}
