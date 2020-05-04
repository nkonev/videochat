package com.github.nkonev.blog.config;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.DefaultStringRedisConnection;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.connection.RedisServerCommands;
import javax.annotation.PostConstruct;

@Configuration
public class UtConfig {

    @Autowired
    private RedisServerCommands redisServerCommands;

    @Bean(destroyMethod = "close")
    public DefaultStringRedisConnection defaultStringRedisConnection(RedisConnectionFactory redisConnectionFactory){
        return new DefaultStringRedisConnection(redisConnectionFactory.getConnection());
    }

    @PostConstruct
    public void dropRedis(){
        redisServerCommands.flushDb();
    }
}
