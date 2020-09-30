package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import static com.github.nkonev.aaa.config.RedisConfig.JSON_REDIS_TEMPLATE;

@Service
public class NotifierService {

    public static final String USER_PROFILE_UPDATE = "user.profile.update";

    @Autowired
    @Qualifier(JSON_REDIS_TEMPLATE)
    private RedisTemplate<String, Object> redisTemplate;

    public void notifyProfileUpdated(UserAccount userAccount) {
        redisTemplate.convertAndSend(USER_PROFILE_UPDATE, UserAccountConverter.convertToUserAccountDTO(userAccount));
    }
}
