package name.nkonev.aaa.dto;

public record SearchUsersRequestDTO(
    int size,
    Long startingFromItemId,
    boolean reverse,
    boolean hasHash,
    String searchString
) {
}
