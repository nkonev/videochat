package name.nkonev.aaa.dto;

/**
 * Slice of FieldError
 */
public record ValidationError (
    String field,
    Object rejectedValue,
    String message
) { }
