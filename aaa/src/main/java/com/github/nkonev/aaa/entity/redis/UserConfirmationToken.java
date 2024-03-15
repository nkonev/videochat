package com.github.nkonev.aaa.entity.redis;

import org.springframework.data.annotation.Id;
import org.springframework.data.redis.core.RedisHash;
import org.springframework.data.redis.core.TimeToLive;

import java.util.UUID;

@RedisHash
public record UserConfirmationToken (
    @Id
    UUID uuid,

    Long userId,

    @TimeToLive
    long ttlSeconds

) { }
