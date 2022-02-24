package com.github.nkonev.integration;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Service;

import java.util.Date;
import java.util.Map;

import static com.github.nkonev.aaa.it.OAuth2EmulatorTests.*;

@Service
public class UserTestService {

    @Autowired
    private NamedParameterJdbcTemplate namedParameterJdbcTemplate;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserTestService.class);

    public void clearOauthBindingsInDb() {
//        var r = "retired_" + new Date().getTime();
//        final var deleteUsersSql = "UPDATE users SET username = :new_username, vkontakte_id=NULL, facebook_id=NULL, google_id=NULL WHERE username = :username";
//        int fb = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", facebookLogin, "new_username", "un_fb_" + r));
//        int vk = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", vkontakteLogin, "new_username", "un_vk_" + r));
//        int go = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", googleLogin, "new_username", "un_go_" + r));
//        LOGGER.info("Renamed {} fb, {} vk, {} google oauth2 users", fb, vk, go);

        final var deleteUsersSql = "DELETE FROM users WHERE username = :username";
        int fb = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", facebookLogin));
        int vk = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", vkontakteLogin));
        int go = namedParameterJdbcTemplate.update(deleteUsersSql, Map.of("username", googleLogin));
        LOGGER.info("Removed {} fb, {} vk, {} google oauth2 users", fb, vk, go);

        int upd = namedParameterJdbcTemplate.update("UPDATE users SET vkontakte_id=NULL, facebook_id=NULL, google_id=NULL", Map.of());
        LOGGER.info("Updated {} oauth2 users", upd);
    }
}
