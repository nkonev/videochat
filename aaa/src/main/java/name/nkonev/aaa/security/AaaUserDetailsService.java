package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserOnlineResponse;
import name.nkonev.aaa.exception.DataNotFoundInternalException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.session.Session;
import org.springframework.session.data.redis.RedisIndexedSessionRepository;
import org.springframework.stereotype.Service;

import java.time.Clock;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.stream.StreamSupport;

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

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

    /**
     * load UserAccountDetailsDTO from database, or throws UsernameNotFoundException
     * @param username
     * @return
     * @throws UsernameNotFoundException
     */
    @Override
    public UserAccountDetailsDTO loadUserByUsername(String username) throws UsernameNotFoundException {
        var ud = userAccountRepository
                .findByUsername(username)
                .map(userAccountConverter::convertToUserAccountDetailsDTO)
                .orElseThrow(() -> new UsernameNotFoundException("User with login '" + username + "' not found"));

        final var now = getNowUTC();
        if (ud.getLastSeenDateTime() == null || now.minus(aaaProperties.onlineEstimation()).isAfter(ud.getLastSeenDateTime())) {
            userAccountRepository.updateLastSeen(username, now);
            ud = ud.withUserAccountDTO(ud.userAccountDTO().withLastSeenDateTime(now));
            eventService.notifyOnlineChanged(List.of(new UserOnlineResponse(ud.getId(), true, now)));
        }
        return ud;
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
        return StreamSupport.stream(userAccountRepository.findAllById(userIds).spliterator(), false)
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.username())), u.lastSeenDateTime()))
                .toList();
    }

    public List<UserOnlineResponse> getUsersOnlineByUsers(List<UserAccount> users){
        if (users == null){
            throw new RuntimeException("users cannon be null");
        }
        return users.stream()
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.username())), u.lastSeenDateTime()))
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
        var userName = userToFillSessions.username();
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
        return getSessions(userAccount.username());
    }
}
