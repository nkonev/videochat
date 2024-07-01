package name.nkonev.aaa.entity.redis;

import org.springframework.data.annotation.Id;
import org.springframework.data.redis.core.RedisHash;
import org.springframework.data.redis.core.TimeToLive;

import java.util.UUID;

@RedisHash
public record UserConfirmationToken (
    @Id
    UUID uuid,

    Long userId,

    String referrer,

    @TimeToLive
    long ttlSeconds

) { }
