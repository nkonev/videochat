package com.github.nkonev.aaa.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisKeyValueAdapter;
import org.springframework.data.redis.repository.configuration.EnableRedisRepositories;

@Configuration
// https://stackoverflow.com/questions/41693774/spring-redis-indexes-not-deleted-after-main-entry-expires/41695902#41695902
@EnableRedisRepositories(basePackages= "com.github.nkonev.aaa.repository.redis", enableKeyspaceEvents=RedisKeyValueAdapter.EnableKeyspaceEvents.ON_STARTUP)
public class RedisConfig {

}
