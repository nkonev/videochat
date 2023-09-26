package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.services.UserTestService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import javax.annotation.PostConstruct;
import java.util.List;

@RestController
@ConditionalOnProperty("custom.user.test")
public class UserTestController {

    @Autowired
    private UserTestService userTestService;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserTestController.class);

    @PostConstruct
    public void pc() {
        LOGGER.warn("Is enabled");
    }

    @PutMapping(Constants.Urls.INTERNAL_API + "/reset")
    public void reset(@RequestBody List<String> users) {
        userTestService.clearOauthBindingsInDb(users);
    }
}
