package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.controllers.UserProfileController;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.exception.DataNotFoundException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.session.FindByIndexNameSessionRepository;
import org.springframework.session.Session;
import org.springframework.session.data.redis.RedisIndexedSessionRepository;
import org.springframework.stereotype.Component;
import org.springframework.util.Assert;

import java.time.Clock;
import java.time.Duration;
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

    @Value("${custom.online-estimation}")
    private Duration onlineEstimatedDuration;

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
                .map(UserAccountConverter::convertToUserAccountDetailsDTO)
                .orElseThrow(() -> new UsernameNotFoundException("User with login '" + username + "' not found"));
    }

    /**
     * Set new UserDetails to SecurityContext.
     * When spring mvc finishes request processing, UserDetails will be stored in Session and effectively appears in Redis
     * @param userAccount
     */
    public void refreshUserDetails(UserAccount userAccount) {
        Assert.notNull(userAccount, "userAccount cannot be null");
        UserAccountDetailsDTO newUd = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        Authentication authentication = new UsernamePasswordAuthenticationToken(newUd, newUd.getPassword(), newUd.getAuthorities());
        Assert.notNull(SecurityContextHolder.getContext(), "securityContext cannot be null");
        SecurityContextHolder.getContext().setAuthentication(authentication);
    }

    public Map<String, Session> getSessions(String userName){
        Object o = redisOperationsSessionRepository.findByIndexNameAndIndexValue(FindByIndexNameSessionRepository.PRINCIPAL_NAME_INDEX_NAME, userName);
        return (Map<String, Session>)o;
    }

    public Map<String, Session> getMySessions(UserDetails userDetails){
        if (userDetails == null){
            throw new RuntimeException("getMySessions may be called only by authorized users");
        }
        return getSessions(userDetails.getUsername());
    }

    public List<UserProfileController.UserOnlineResponse> getUsersOnline(List<Long> userIds){
        if (userIds == null){
            throw new RuntimeException("userIds cannon be null");
        }
        return StreamSupport.stream(userAccountRepository.findAllById(userIds).spliterator(), false)
                .map(u -> new UserProfileController.UserOnlineResponse(u.id(), calcOnline(getSessions(u.username()))))
                .toList();
    }

    public List<UserProfileController.UserOnlineResponse> getUsersOnlineByUsers(List<UserAccount> users){
        if (users == null){
            throw new RuntimeException("users cannon be null");
        }
        return users.stream()
                .map(u -> new UserProfileController.UserOnlineResponse(u.id(), calcOnline(getSessions(u.username()))))
                .toList();
    }

    private boolean calcOnline(Map<String, Session> sessions) {
        return sessions.entrySet().stream().anyMatch(session -> {
            return session.getValue().getLastAccessedTime().plus(onlineEstimatedDuration).isAfter(Instant.now(Clock.systemUTC()));
        });
    }

    public UserAccount getUserAccount(long userId){
        return userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundException("User with id " + userId + " not found"));
    }

    public void killSessions(long userId){
        String userName = getUserAccount(userId).username();
        Map<String, Session> sessionMap = getSessions(userName);
        sessionMap.keySet().forEach(session -> redisOperationsSessionRepository.deleteById(session));
    }

    public void killSessions(String userName){
        Map<String, Session> sessionMap = getSessions(userName);
        sessionMap.keySet().forEach(session -> redisOperationsSessionRepository.deleteById(session));
    }

    public Map<String, Session> getSessions(long userId) {
        UserAccount userAccount = getUserAccount(userId);
        return getSessions(userAccount.username());
    }
}
