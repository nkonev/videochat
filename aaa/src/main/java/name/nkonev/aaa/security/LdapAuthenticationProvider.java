package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.ldap.LdapEntity;
import name.nkonev.aaa.exception.OAuth2IdConflictException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.services.ConflictResolvingActions;
import name.nkonev.aaa.services.ConflictService;
import name.nkonev.aaa.services.EventService;
import name.nkonev.aaa.services.tasks.LdapSyncRolesService;
import name.nkonev.aaa.utils.PageUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.dao.IncorrectResultSizeDataAccessException;
import org.springframework.ldap.core.LdapOperations;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.security.authentication.AuthenticationProvider;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;

import static name.nkonev.aaa.Constants.LDAP_CONFLICT_PREFIX;
import static name.nkonev.aaa.converter.UserAccountConverter.validateLengthAndTrimLogin;
import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

// https://spring.io/guides/gs/authenticating-ldap
@Component
public class LdapAuthenticationProvider implements AuthenticationProvider, ConflictResolvingActions {

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Autowired
    private ConflictService conflictService;

    @Autowired
    private LdapSyncRolesService ldapSyncRolesService;

    private static final Logger LOGGER = LoggerFactory.getLogger(LdapAuthenticationProvider.class);

    @Override
    public Authentication authenticate(Authentication authentication) throws AuthenticationException {
        if (aaaProperties.ldap().auth().enabled()) {
            UsernamePasswordAuthenticationToken usernamePasswordAuthenticationToken = (UsernamePasswordAuthenticationToken) authentication;
            var userName = validateLengthAndTrimLogin(usernamePasswordAuthenticationToken.getPrincipal().toString(), true);
            var password = usernamePasswordAuthenticationToken.getCredentials().toString();

            var encodedPassword = encodePassword(password);

            try {
                List<EventWrapper<?>> eventsContainer = new ArrayList<>();
                var userAccount = transactionTemplate.execute(status -> {
                    var lq = LdapQueryBuilder.query().base(aaaProperties.ldap().auth().base()).filter(aaaProperties.ldap().auth().filter(), userName);
                    ldapOperations.authenticate(lq, encodedPassword);

                    var ldapAttributes = ldapOperations.searchForContext(lq).getAttributes();
                    var ldapEntry = new LdapEntity(aaaProperties.ldap().attributeNames(), ldapAttributes);

                    var ldapUserId = ldapEntry.id();
                    if (ldapUserId == null) {
                        LOGGER.warn("Got null ldap id for username={}", userName);
                        return null;
                    }

                    return userAccountRepository
                        .findByLdapId(ldapUserId) // find an existing user
                        .orElseGet(() -> {
                            // or try to create a new user
                            String email = null;
                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().email())) {
                                email = ldapEntry.email();
                            }

                            Set<String> rawRoles = new HashSet<>();
                            if (aaaProperties.schedulers().syncLdap().syncRoles()) {
                                var extAdminRole = ldapSyncRolesService.getNecessaryAdminRole();
                                ldapSyncRolesService.processRoles(PageUtils.DEFAULT_SIZE, extAdminRole, batch -> {
                                    for (var userRole : batch) {
                                        if (userRole.id().equals(ldapUserId)) {
                                            rawRoles.add(extAdminRole);
                                            break;
                                        }
                                    }
                                });
                            }
                            var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().ldap(), rawRoles);

                            var userToInsert = userAccountConverter.buildUserAccountEntityForLdapInsert(
                                    userName,
                                    ldapUserId,
                                    mappedRoles,
                                    email,
                                    false,
                                    true,
                                    getNowUTC()
                            );
                            // check for conflicts by username or email and create the user if conflict resolution is not "ignore"
                            conflictService.process(LDAP_CONFLICT_PREFIX, aaaProperties.ldap().resolveConflictsStrategy(), ConflictService.PotentiallyConflictingAction.INSERT, userToInsert, this, eventsContainer);
                            // due to conflict we can ignore the user and not to save him or we can create a new
                            // so we try to lookup him
                            var foundNewUser = userAccountRepository.findByLdapId(ldapUserId)
                                    .orElseThrow(() -> new OAuth2IdConflictException(("User with ldapId = " + ldapUserId + " is not found after conflict solving")));
                            return foundNewUser;
                        });
                });
                if (userAccount == null) {
                    LOGGER.info("Skipping login via ldap by username {}", userName);
                    return null;
                }
                UserAccountDetailsDTO userDetails = userAccountConverter.convertToUserAccountDetailsDTO(userAccount);

                sendEvents(eventsContainer);

                return new AaaAuthenticationToken(userDetails);
            } catch (IncorrectResultSizeDataAccessException e) {
                LOGGER.debug("Unable to authenticate via LDAP", e);
            }
        }
        return null;
    }

    private String encodePassword(String password) {
        switch (aaaProperties.ldap().password().encodingType().toLowerCase()){
            case "bcrypt":
                return new BCryptPasswordEncoder(aaaProperties.ldap().password().strength()).encode(password);
            default:
                return password;
        }
    }

    @Override
    public boolean supports(Class<?> authentication) {
        return (UsernamePasswordAuthenticationToken.class.isAssignableFrom(authentication));
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
