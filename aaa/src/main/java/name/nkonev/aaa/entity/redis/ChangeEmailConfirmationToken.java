package name.nkonev.aaa.entity.redis;

import org.springframework.data.annotation.Id;
import org.springframework.data.redis.core.RedisHash;
import org.springframework.data.redis.core.TimeToLive;

import java.util.UUID;

@RedisHash
public record ChangeEmailConfirmationToken(
    @Id
    Long userId,

    UUID uuid,

    String newEmail,

    @TimeToLive
    long ttlSeconds

) { }
