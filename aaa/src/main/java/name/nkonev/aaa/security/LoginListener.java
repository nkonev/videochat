package name.nkonev.aaa.security;

import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.dto.UserOnlineResponse;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.ApplicationListener;
import org.springframework.security.authentication.event.AuthenticationSuccessEvent;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

/**
 * This listener works for both database and OAuth2 logins
 */
@Transactional
@Component
public class LoginListener implements ApplicationListener<AuthenticationSuccessEvent> {

    private static final Logger LOGGER = LoggerFactory.getLogger(LoginListener.class);

    @Autowired
    private EventService eventService;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Override
    public void onApplicationEvent(AuthenticationSuccessEvent event) {
        UserAccountDetailsDTO userDetails = (UserAccountDetailsDTO) event.getAuthentication().getPrincipal();
        onApplicationEvent(userDetails);
    }

    public void onApplicationEvent(UserAccountDetailsDTO userDetails) {
        LOGGER.info("User '{}' logged in", userDetails.getUsername());
        final var now = getNowUTC();
        userAccountRepository.updateLastSeen(userDetails.getUsername(), now);
        eventService.notifyOnlineChanged(List.of(new UserOnlineResponse(userDetails.getId(), true, now)));
    }

}
