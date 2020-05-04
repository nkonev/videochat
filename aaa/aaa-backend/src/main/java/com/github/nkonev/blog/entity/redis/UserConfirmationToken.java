package com.github.nkonev.blog.entity.redis;

import org.springframework.data.annotation.Id;
import org.springframework.data.redis.core.RedisHash;
import org.springframework.data.redis.core.TimeToLive;

@RedisHash
public class UserConfirmationToken {
    @Id
    private String uuid;

    private Long userId;

    @TimeToLive
    private long ttlSeconds;

    public UserConfirmationToken() { }

    public UserConfirmationToken(String uuid, Long userId, long ttlSeconds) {
        this.uuid = uuid;
        this.userId = userId;
        this.ttlSeconds = ttlSeconds;
    }

    public String getUuid() {
        return uuid;
    }

    public void setUuid(String uuid) {
        this.uuid = uuid;
    }

    public Long getUserId() {
        return userId;
    }

    public void setUserId(Long userId) {
        this.userId = userId;
    }

    public long getTtlSeconds() {
        return ttlSeconds;
    }

    public void setTtlSeconds(long ttlSeconds) {
        this.ttlSeconds = ttlSeconds;
    }
}
