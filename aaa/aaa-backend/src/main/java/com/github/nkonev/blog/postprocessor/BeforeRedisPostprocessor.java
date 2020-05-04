package com.github.nkonev.blog.postprocessor;

import com.github.nkonev.blog.services.port.RedisPortChecker;
import org.springframework.boot.autoconfigure.AbstractDependsOnBeanFactoryPostProcessor;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;

@Configuration
public class BeforeRedisPostprocessor extends AbstractDependsOnBeanFactoryPostProcessor {

	public BeforeRedisPostprocessor() {
        super(RedisConnectionFactory.class, RedisPortChecker.NAME);
    }
}
