package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.exception.OAuth2IdConflictException;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.exception.UserConflictException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import name.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import name.nkonev.aaa.services.ConflictResolvingActions;
import name.nkonev.aaa.services.ConflictService;
import name.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.Assert;

import java.util.*;

import static name.nkonev.aaa.converter.UserAccountConverter.validateLengthAndTrimLogin;
import static name.nkonev.aaa.utils.ServletUtils.getCurrentHttpRequest;

public abstract class AbstractOAuth2UserService implements ConflictResolvingActions {

    @Autowired
    private AaaPreAuthenticationChecks aaaPreAuthenticationChecks;

    @Autowired
    private AaaPostAuthenticationChecks aaaPostAuthenticationChecks;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    @Autowired
    protected UserAccountConverter userAccountConverter;

    @Autowired
    private ConflictService conflictService;

    @Autowired
    protected UserAccountRepository userAccountRepository;

    private boolean isAlreadyAuthenticated(){
        return SecurityContextHolder.getContext().getAuthentication()!=null && SecurityContextHolder.getContext().getAuthentication().getPrincipal() instanceof UserAccountDetailsDTO;
    }

    private UserAccountDetailsDTO getPrincipal() {
        return SecurityUtils.getPrincipal();
    }

    protected abstract Logger logger();

    protected abstract String getOAuth2Name();

    protected abstract Optional<UserAccount> findByOAuth2Id(String oauthId);

    protected abstract UserAccountDetailsDTO setOAuth2IdToPrincipal(UserAccountDetailsDTO principal, String oauthId);

    protected abstract UserAccount setOAuth2IdToEntity(Long id, String oauthId);

    protected abstract UserAccount buildEntity(String oauthId, String login, Map<String, Object> oauthResourceServerResponse, Set<String> roles);

    // @return notnull UserAccountDetailsDTO if was merged - so we should return it immediately from Extractor
    private UserAccountDetailsDTO mergeOAuth2IdToCurrentUser(String oauthId, List<EventWrapper<?>> eventsContainer) {
        if (isAlreadyAuthenticated()) {
            // we already authenticated - so it' s binding (setting oauth2_id)
            UserAccountDetailsDTO principal = getPrincipal();
            logger().info("Will merge {}Id to exists user '{}', id={}", getOAuth2Name(), principal.getUsername(), principal.getId());

            Optional<UserAccount> maybeUserAccount = findByOAuth2Id(oauthId);
            if (maybeUserAccount.isPresent() && !maybeUserAccount.get().id().equals(principal.getId())){
                logger().info("With {}Id={} already present another user '{}', id={}", getOAuth2Name(), oauthId, maybeUserAccount.get().login(), maybeUserAccount.get().id());
                throw new OAuth2IdConflictException("Somebody already taken this "+ getOAuth2Name()+" id="+oauthId+". " +
                        "If this is you and you want to merge your profiles please delete another profile and bind "+ getOAuth2Name()+" to this. If not please contact administrator.");
            }

            principal = setOAuth2IdToPrincipal(principal, oauthId);
            var userAccount = setOAuth2IdToEntity(principal.getId(), oauthId);
            eventsContainer.add(eventService.convertProfileUpdated(userAccount));

            logger().info("{}Id successfully merged to exists user '{}', id={}", getOAuth2Name(), principal.getUsername(), principal.getId());

            boolean setToSession = false;
            var request = getCurrentHttpRequest();
            if (request != null) {
                var session = request.getSession(false);
                if (session != null) {
                    SecurityUtils.setToContext(session, principal);
                    setToSession = true;
                }
            }
            if (!setToSession) {
                logger().warn("Unable to set changed principal to session");
            }

            return principal;
        } else {
            return null;
        }
    }


    private UserAccountDetailsDTO createOrGetExistingUser(String oauthId, String login0, Map<String, Object> attributes, Set<String> roles, List<EventWrapper<?>> eventsContainer) {
        UserAccount userAccount;
        Optional<UserAccount> userAccountOpt = findByOAuth2Id(oauthId);
        if (userAccountOpt.isEmpty()) { // we didn't find an user account by oauth_id
            var login = validateLengthAndTrimLogin(login0, true);
            var newUserAccount = buildEntity(oauthId, login, attributes, roles);

            // insert (optionally with conflict solving)
            conflictService.process(getConflictPrefix(), getConflictResolveStrategy(), ConflictService.PotentiallyConflictingAction.INSERT, newUserAccount, this, eventsContainer);
            // due to conflict we can ignore the user and not to save him or we can create a new
            // so we try to lookup him
            userAccount = findByOAuth2Id(oauthId)
                    .orElseThrow(() -> new UserConflictException(("User with "+getOAuth2Name()+"Id = " + oauthId + " is not found after conflict solving")));
        } else { // get existing
            userAccount = userAccountOpt.get();
        }

        return userAccountConverter.convertToUserAccountDetailsDTO(userAccount);
    }

    abstract String getConflictPrefix();

    abstract ConflictResolveStrategy getConflictResolveStrategy();

    abstract protected String getLogin(Map<String, Object> map);

    abstract protected String getId(Map<String, Object> map);

    protected UserAccountDetailsDTO process(Map<String, Object> map, OAuth2UserRequest userRequest) {
        String oauth2userId = getId(map);
        Assert.hasLength(oauth2userId, getOAuth2Name() + " id cannot be empty");

        List<EventWrapper<?>> eventsContainer = new ArrayList<>();
        UserAccountDetailsDTO principal = transactionTemplate.execute(status -> {
            UserAccountDetailsDTO resultPrincipal;

            var mergeOAuthToUserResponse = mergeOAuth2IdToCurrentUser(oauth2userId, eventsContainer);
            if (mergeOAuthToUserResponse != null) { // was authenticated and merge happened
                // ok
                resultPrincipal = mergeOAuthToUserResponse;
            } else {
                // wasn't authenticated so we return null and here we are going to create the new user account
                String login = getLogin(map);
                resultPrincipal = createOrGetExistingUser(oauth2userId, login, map, getRoles(userRequest), eventsContainer);
            }

            aaaPreAuthenticationChecks.check(resultPrincipal);
            aaaPostAuthenticationChecks.check(resultPrincipal);
            return resultPrincipal;
        });
        sendEvents(eventsContainer);

        return principal;
    }

    protected Set<String> getRoles(OAuth2UserRequest userRequest) {
        return null;
    }

    @Override
    public void insertUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer) {
        var saved = userAccountRepository.save(userAccount);
        eventsContainer.add(eventService.convertProfileCreated(saved));
    }

    @Override
    public void updateUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer) {
        var updated = userAccountRepository.save(userAccount);
        eventsContainer.add(eventService.convertProfileUpdated(updated));
    }

    @Override
    public void insertUsers(Collection<UserAccount> users, List<EventWrapper<?>> eventsContainer) {
        var saved = userAccountRepository.saveAll(users);
        for (UserAccount userAccount : saved) {
            eventsContainer.add(eventService.convertProfileCreated(userAccount));
        }
    }

    @Override
    public void updateUsers(Collection<UserAccount> users, List<EventWrapper<?>> eventsContainer) {
        for (UserAccount userAccount : users) {
            eventsContainer.add(eventService.convertProfileUpdated(userAccount));
        }
        userAccountRepository.saveAll(users);
    }

    @Override
    public void removeUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer) {
        userAccountRepository.deleteById(userAccount.id());
        eventsContainer.add(eventService.convertProfileDeleted(userAccount.id()));
    }

    protected void sendEvents(List<EventWrapper<?>> events) {
        for (EventWrapper<?> event : events) {
            eventService.sendProfileEvent(event);
        }
        events.clear();
    }
}

