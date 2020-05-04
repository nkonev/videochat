package com.github.nkonev.blog.services.port;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.data.redis.RedisProperties;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.stereotype.Service;

import javax.annotation.PostConstruct;
import java.net.URI;

@Service(RedisPortChecker.NAME)
public class RedisPortChecker extends AbstractPortChecker{

    public static final String NAME="redisPortChecker";

    @Value("${port.check.redis.max.count:64}")
    private int maxCount;

    @Autowired
    private RedisProperties redisProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(RedisPortChecker.class);

    @PostConstruct
    public void checkPorts(){
        LOGGER.info("Will check redis connection");
        URI uri = URI.create(redisProperties.getUrl());
        check(maxCount, uri.getHost(), uri.getPort());
        LOGGER.info("Redis connection is ok");
    }

    @Override
    protected Logger getLogger() {
        return LOGGER;
    }
}
