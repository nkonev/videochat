package com.github.nkonev.aaa.services;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Service;
import java.util.List;
import java.util.Map;


@Service
@ConditionalOnProperty("custom.user.test")
public class UserTestService {

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserTestService.class);

    public void clearOauthBindingsInDb(List<String> logins) {
        final var deleteUsersSql = "DELETE FROM users WHERE username = :username";
        for (var login: logins) {
            int updated = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", login));
            LOGGER.info("Removed {} {} oauth2 user", updated, login);
        }
        int upd = namedParameterJdbcTemplate.update("UPDATE users SET vkontakte_id=NULL, facebook_id=NULL, google_id=NULL", Map.of());
        LOGGER.info("Updated {} oauth2 users", upd);
    }
}
