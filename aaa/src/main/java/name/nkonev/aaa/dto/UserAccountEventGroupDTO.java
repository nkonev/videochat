package name.nkonev.aaa.dto;

public record UserAccountEventGroupDTO(
    Long userId,
    String eventType,
    UserAccountDTOExtended forMyself,
    UserAccountDTOExtended forRoleAdmin,
    UserAccountDTO forRoleUser
) { }
