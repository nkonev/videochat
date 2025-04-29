package name.nkonev.aaa.dto;


import java.time.LocalDateTime;

public record UserOnlineResponse(
        long userId,
        boolean online,
        LocalDateTime lastSeenDateTime
) { }
