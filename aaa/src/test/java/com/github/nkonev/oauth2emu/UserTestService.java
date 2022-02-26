package com.github.nkonev.oauth2emu;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Service;
import java.util.Map;

import static com.github.nkonev.aaa.it.OAuth2EmulatorTests.*;

@Service
public class UserTestService {

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserTestService.class);

    public void clearOauthBindingsInDb() {
        final var deleteUsersSql = "DELETE FROM users WHERE username = :username";
        int fb = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", facebookLogin));
        int vk = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", vkontakteLogin));
        int go = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", googleLogin));
        LOGGER.info("Removed {} fb, {} vk, {} google oauth2 users", fb, vk, go);

        int upd = namedParameterJdbcTemplate.update("UPDATE users SET vkontakte_id=NULL, facebook_id=NULL, google_id=NULL", Map.of());
        LOGGER.info("Updated {} oauth2 users", upd);
    }
}
