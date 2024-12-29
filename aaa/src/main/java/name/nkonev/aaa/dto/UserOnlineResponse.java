package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import name.nkonev.aaa.Constants;

import java.time.LocalDateTime;

public record UserOnlineResponse(
        long userId,
        boolean online,
        @JsonFormat(shape=JsonFormat.Shape.STRING, pattern= Constants.DATE_FORMAT)
        LocalDateTime lastSeenDateTime
) { }
