package name.nkonev.aaa.dto;

public record SearchUsersRequestDTO(
    int size,
    Long startingFromItemId,
    boolean includeStartingFrom,
    boolean reverse,
    String searchString
) {
}
