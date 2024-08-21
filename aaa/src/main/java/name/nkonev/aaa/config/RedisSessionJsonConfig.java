package name.nkonev.aaa.config;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonTypeInfo;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.springframework.beans.factory.BeanClassLoaderAware;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.serializer.GenericJackson2JsonRedisSerializer;
import org.springframework.data.redis.serializer.RedisSerializer;
import org.springframework.security.jackson2.SecurityJackson2Modules;

import java.util.HashSet;

@Configuration
public class RedisSessionJsonConfig implements BeanClassLoaderAware {

    private ClassLoader loader;

    ObjectMapper objectMapper() {
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModules(SecurityJackson2Modules.getModules(this.loader));
        mapper.registerModule(new JavaTimeModule());
        mapper.addMixIn(Long.class, LongMixin.class);
        mapper.addMixIn(new HashSet<>().getClass(), HashSetMixin.class);
        return mapper;
    }

    // bean name used from RedisHttpSessionConfiguration#setDefaultRedisSerializer
    @Bean(name = "springSessionDefaultRedisSerializer")
    public RedisSerializer<Object> redisSerializer() {
        return new GenericJackson2JsonRedisSerializer(objectMapper());
    }

    @Override
    public void setBeanClassLoader(ClassLoader classLoader) {
        this.loader = classLoader;
    }
}

// from https://github.com/spring-projects/spring-session/issues/2227#issuecomment-1418932679
// wait for https://github.com/spring-projects/spring-session/issues/2305
abstract class LongMixin {
    @SuppressWarnings("unused")
    @JsonProperty("long")
    Long value;
}

@JsonTypeInfo(use = JsonTypeInfo.Id.CLASS, include = JsonTypeInfo.As.PROPERTY, property = "@class")
abstract class HashSetMixin {
}
