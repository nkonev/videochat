package com.github.nkonev.blog.entity.redis;

import org.springframework.data.redis.core.RedisHash;

import org.springframework.data.annotation.Id;
import org.springframework.data.redis.core.TimeToLive;

@RedisHash
public class PasswordResetToken {
    @Id
    private String uuid;

    private Long userId;

    @TimeToLive
    private long ttlSeconds;

    public PasswordResetToken() { }

    public PasswordResetToken(String uuid, Long userId, long ttlSeconds) {
        this.uuid = uuid;
        this.userId = userId;
        this.ttlSeconds = ttlSeconds;
    }

    public Long getUserId() {
        return userId;
    }

    public void setUserId(Long userId) {
        this.userId = userId;
    }

    public String getUuid() {
        return uuid;
    }

    public void setUuid(String uuid) {
        this.uuid = uuid;
    }

    public long getTtlSeconds() {
        return ttlSeconds;
    }

    public void setTtlSeconds(long ttlSeconds) {
        this.ttlSeconds = ttlSeconds;
    }
}
