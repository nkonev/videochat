package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserOnlineResponse;
import name.nkonev.aaa.exception.DataNotFoundException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.services.EventService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.session.Session;
import org.springframework.session.data.redis.RedisIndexedSessionRepository;
import org.springframework.stereotype.Component;

import java.time.Clock;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.stream.StreamSupport;

/**
 * Provides Spring Security compatible UserAccountDetailsDTO.
 */
@Component
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

    /**
     * load UserAccountDetailsDTO from database, or throws UsernameNotFoundException
     * @param username
     * @return
     * @throws UsernameNotFoundException
     */
    @Override
    public UserAccountDetailsDTO loadUserByUsername(String username) throws UsernameNotFoundException {
        return userAccountRepository
                .findByUsername(username)
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

    private record UsernameWithId(String username, Long id){}

    public List<UserOnlineResponse> getUsersOnline(List<Long> userIds){
        if (userIds == null){
            throw new RuntimeException("userIds cannon be null");
        }
        return userIds.stream().map(uid -> new UsernameWithId(UserAccountDetailsDTO.toUsername(uid), uid))
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.username()))))
                .toList();
    }

    public List<UserOnlineResponse> getUsersOnlineByUsers(List<UserAccount> users){
        if (users == null){
            throw new RuntimeException("users cannon be null");
        }
        return users.stream()
                .map(u -> new UserOnlineResponse(u.id(), calcOnline(getSessions(u.username()))))
                .toList();
    }

    private boolean calcOnline(Map<String, Session> sessions) {
        return sessions.entrySet().stream().anyMatch(session -> {
            return session.getValue().getLastAccessedTime().plus(aaaProperties.onlineEstimation()).isAfter(Instant.now(Clock.systemUTC()));
        });
    }

    public UserAccount getUserAccount(long userId){
        return userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundException("User with id " + userId + " not found"));
    }

    public void killSessions(long userId, ForceKillSessionsReasonType reasonType) {
        killSessions(userId, reasonType, null, null);
    }

    public void killSessions(long userId, ForceKillSessionsReasonType reasonType, String filterOutSession, Long currentUserId){
        String userName = getUserAccount(userId).username();
        Map<String, Session> sessionMap = getSessions(userName);
        sessionMap.keySet().stream().filter(aSession -> filterOutSession != null ? !aSession.equals(filterOutSession) : true).forEach(session -> redisOperationsSessionRepository.deleteById(session));

        if (currentUserId != null && currentUserId.equals(userId)){
            // nothing
        } else {
            eventService.notifySessionsKilled(userId, reasonType);
        }
    }

    public Map<String, Session> getSessions(long userId) {
        UserAccount userAccount = getUserAccount(userId);
        return getSessions(userAccount.username());
    }
}
