package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.controllers.UserProfileController;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.ApplicationListener;
import org.springframework.security.authentication.event.AuthenticationSuccessEvent;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;
import static com.github.nkonev.aaa.utils.TimeUtil.getNowUTC;

/**
 * This listener works for both database and OAuth2 logins
 */
@Transactional
@Component
public class LoginListener implements ApplicationListener<AuthenticationSuccessEvent> {

    private static final Logger LOGGER = LoggerFactory.getLogger(LoginListener.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private EventService eventService;

    @Override
    public void onApplicationEvent(AuthenticationSuccessEvent event) {
        UserAccountDetailsDTO userDetails = (UserAccountDetailsDTO) event.getAuthentication().getPrincipal();
        LOGGER.info("User '{}' logged in", userDetails.getUsername());
        userAccountRepository.updateLastLogin(userDetails.getUsername(), getNowUTC());
        eventService.notifyOnlineChanged(new UserProfileController.UserOnlineResponse(userDetails.getId(), true));
    }
}
