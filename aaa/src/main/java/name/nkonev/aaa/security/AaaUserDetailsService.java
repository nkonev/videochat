package name.nkonev.aaa.security;

import java.time.Clock;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.stream.StreamSupport;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.session.Session;
import org.springframework.session.data.redis.RedisIndexedSessionRepository;
import org.springframework.stereotype.Service;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.dto.UserOnlineResponse;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.exception.DataNotFoundInternalException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.services.EventService;

/**
 * Provides Spring Security compatible UserAccountDetailsDTO.
 */
@Service
public class AaaUserDetailsService implements UserDetailsService {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private RedisIndexedSessionRepository redisOperationsSessionRepository;

    @Autowired
    private EventService eventService;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private UserAccountConverter userAccountConverter;

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaUserDetailsService.class);

    public static final String MESSAGE_WITH_EXPOSED_SECRET = "A message with exposed db connection url";

    /**
     * load UserAccountDetailsDTO from database, or throws UsernameNotFoundException
     * @param username
     * @return
     * @throws UsernameNotFoundException
     */
    @Override
    public UserAccountDetailsDTO loadUserByUsername(String username) throws UsernameNotFoundException {
        if (aaaProperties.security().failLoginWithSecretExposing()) {
            // for tests only
            throw new RuntimeException(MESSAGE_WITH_EXPOSED_SECRET);
        }

        return userAccountRepository
                .findByLogin(username)
                .map(userAccountConverter::convertToUserAccountDetailsDTO)
                .orElseThrow(() -> new UsernameNotFoundException("User with login '" + username + "' not found"));
    }

    public Map<String, Session> getSessions(String userName){
        Object o = redisOperationsSessionRepository.findByPrincipalName(userName);
        return (Map<String, Session>)o;
    }

    public Map<String, Session> getMySessions(UserDetails userDetails){
        if (userDetails == null){
            throw new RuntimeException("getMySessions may be called only by authorized users");
        }
        return getSessions(userDetails.getUsername());
    }

    public List<UserOnlineResponse> getUsersOnline(List<Long> userIds){
        if (userIds == null){
            throw new RuntimeException("userIds cannon be null");
        }
        LOGGER.info("is online");
        return StreamSupport.stream(userAccountRepository.findAllById(userIds).spliterator(), false)
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.login())), u.lastSeenDateTime()))
                .toList();
    }

    public List<UserOnlineResponse> getUsersOnlineByUsers(List<UserAccount> users){
        if (users == null){
            throw new RuntimeException("users cannon be null");
        }
        return users.stream()
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.login())), u.lastSeenDateTime()))
                .toList();
    }

    private boolean calcOnline(Map<String, Session> sessions) {
        return sessions.entrySet().stream().anyMatch(session -> {
            return session.getValue().getLastAccessedTime().plus(aaaProperties.onlineEstimation()).isAfter(Instant.now(Clock.systemUTC()));
        });
    }

    public UserAccount getUserAccount(long userId){
        return userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundInternalException("User with id " + userId + " not found"));
    }

    public void killSessions(long userId, ForceKillSessionsReasonType reasonType) {
        killSessions(userId, reasonType, null, null);
    }

    public void killSessions(UserAccount userToFillSessions, ForceKillSessionsReasonType reasonType) {
        killSessions(userToFillSessions, reasonType, null, null);
    }

    public void killSessions(long userId, ForceKillSessionsReasonType reasonType, String filterOutSession, Long currentUserId){
        UserAccount userToFillSessions = getUserAccount(userId);
        killSessions(userToFillSessions, reasonType, filterOutSession, currentUserId);
    }

    public void killSessions(UserAccount userToFillSessions, ForceKillSessionsReasonType reasonType, String filterOutSession, Long currentUserId){
        var userId = userToFillSessions.id();
        var userName = userToFillSessions.login();
        LOGGER.info("Killing sessions for userId={}, reason={}", userId, reasonType);
        Map<String, Session> sessionMap = getSessions(userName);
        sessionMap.keySet().stream().filter(aSession -> filterOutSession != null ? !aSession.equals(filterOutSession) : true).forEach(session -> redisOperationsSessionRepository.deleteById(session));

        if (currentUserId != null && currentUserId.equals(userId)){
            // nothing
        } else {
            eventService.notifySessionsKilled(userId, reasonType);
            eventService.notifyOnlineChanged(List.of(new UserOnlineResponse(userId, false, userToFillSessions.lastSeenDateTime())));
        }
    }

    public Map<String, Session> getSessions(long userId) {
        UserAccount userAccount = getUserAccount(userId);
        return getSessions(userAccount.login());
    }
}
