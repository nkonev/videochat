package name.nkonev.aaa.dto;

public record EventWrapper<E>(
        E event,
        String type
) {
}
