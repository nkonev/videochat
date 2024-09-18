package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.ldap.LdapEntity;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.security.RoleMapper;
import name.nkonev.aaa.services.ConflictResolvingActions;
import name.nkonev.aaa.services.ConflictService;
import net.javacrumbs.shedlock.spring.annotation.SchedulerLock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ldap.core.*;
import org.springframework.ldap.filter.WhitespaceWildcardsFilter;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.ldap.support.LdapUtils;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.function.Consumer;
import java.util.stream.Collectors;
import javax.naming.NameClassPair;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.directory.DirContext;
import javax.naming.directory.SearchControls;
import javax.naming.directory.SearchResult;

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

@Service
public class SyncLdapTask implements ConflictResolvingActions {
    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private TransactionTemplate transactionTemplate;

    @Autowired
    private ConflictService conflictService;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncLdapTask.class);

    private LocalDateTime currTime;

    @Scheduled(cron = "${custom.schedulers.sync-ldap.cron}")
    @SchedulerLock(name = "syncLdapTask")
    public void scheduledTask() {
        if (!aaaProperties.schedulers().syncLdap().enabled()) {
            return;
        }

        this.doWork();
    }

    public void doWork() {
        LOGGER.info("Sync ldap task start, batchSize={}", aaaProperties.schedulers().syncLdap().batchSize());

        currTime = getNowUTC();

        LOGGER.info("Upserting entries from LDAP");
        var usernameAttrName = aaaProperties.ldap().attributeNames().username();
        var lq = LdapQueryBuilder.query().base(aaaProperties.ldap().auth().base()).filter(new WhitespaceWildcardsFilter(usernameAttrName, " "));

        // partial copy-paste from LdapTemplate because of near Long.MAX_VALUE length of array in spliterator in Spliterators.spliteratorUnknownSize()
        SearchControls controls = new SearchControls();
        controls.setSearchScope(SearchControls.SUBTREE_SCOPE);
        if (lq.searchScope() != null) {
            controls.setSearchScope(lq.searchScope().getId());
        }
        controls.setReturningObjFlag(true);
        SearchExecutor se = (DirContext ctx) -> {
            // var filterValue = "uid=*";
            var filterValue = lq.filter().encode();
            LOGGER.debug("Executing search with base [{}] and filter [{}]", lq.base(), filterValue);
            return ctx.search(lq.base(), filterValue, controls);
        };
        AttributesMapper<LdapEntity> mapper = (Attributes attributes) -> new LdapEntity(aaaProperties.ldap().attributeNames(), attributes);
        MappingConsumingCallbackHandler<LdapEntity> handler = new MappingConsumingCallbackHandler<>(mapper, this::processUserBatch, aaaProperties.schedulers().syncLdap().batchSize());
        ldapOperations.search(se, handler);
        handler.processLeftovers();

        LOGGER.info("Deleting entries from database which were removed from LDAP");
        var deleted = transactionTemplate.execute(s -> userAccountRepository.deleteWithLdapIdElderThan(currTime));
        LOGGER.info("Deleted {} entries from database which were removed from LDAP", deleted);

        LOGGER.info("Sync ldap task finish");
    }

    private void processUserBatch(List<LdapEntity> attributes) {
        transactionTemplate.executeWithoutResult(s -> {
            Map<String, LdapEntity> byLdapId = new HashMap<>();
            for (var ldapEntry : attributes) {
                var ldapUserId = ldapEntry.id();
                if (StringUtils.hasLength(ldapUserId)) {
                    byLdapId.put(ldapUserId, ldapEntry);
                }
            }
            var chunk = userAccountRepository.findByLdapIdInOrderById(byLdapId.keySet());

            var toInsert = new ArrayList<LdapEntity>();
            var toUpdateSyncLdapTime = new HashSet<String>();
            for (var entry : byLdapId.entrySet()) {
                try {
                    var ldapUserId = entry.getKey();
                    var ldapEntry = entry.getValue();
                    LOGGER.info("Examining user with ldapId={}", ldapUserId);

                    if (StringUtils.hasLength(ldapUserId)) {
                        var o = chunk.stream().filter(ua -> ua.ldapId().equals(ldapUserId)).findAny();
                        if (o.isPresent()) { // update the existing user
                            LOGGER.info("User with ldapId={} is existing in database, deciding to update him or not", ldapUserId);
                            var userAccount = o.get();
                            var shouldUpdateInDb = false;

                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().username())) {
                                var ldapUsername = ldapEntry.username();
                                if (StringUtils.hasLength(ldapUsername)) {
                                    if (!ldapUsername.equals(userAccount.username())) {
                                        LOGGER.info("For userId={}, ldapId={}, setting username={}", userAccount.id(), ldapUserId, ldapUsername);
                                        userAccount = userAccount.withUsername(ldapUsername);
                                        shouldUpdateInDb = true;
                                    }
                                } else {
                                    LOGGER.warn("For userId={}, ldapId={}, got empty ldap username", userAccount.id(), ldapUserId);
                                }
                            }

                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().email())) {
                                var ldapEmail = ldapEntry.email();
                                if (!Objects.equals(ldapEmail, userAccount.email())) {
                                    LOGGER.info("For userId={}, ldapId={}, setting email={}", userAccount.id(), ldapUserId, ldapEmail);
                                    userAccount = userAccount.withEmail(ldapEmail);
                                    shouldUpdateInDb = true;
                                }
                            }

                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().role())) {
                                Set<String> rawRoles = ldapEntry.roles();
                                var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().ldap(), rawRoles);
                                var oldRoles = Arrays.stream(userAccount.roles()).collect(Collectors.toSet());
                                if (!oldRoles.equals(mappedRoles)) {
                                    LOGGER.info("For userId={}, ldapId={}, setting roles={}", userAccount.id(), ldapUserId, mappedRoles);
                                    aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_roles_changed);
                                    userAccount = userAccount.withRoles(mappedRoles.toArray(UserRole[]::new));
                                    shouldUpdateInDb = true;
                                }
                            }

                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().locked())) {
                                var ldapLockedV = ldapEntry.locked();
                                if (ldapLockedV != null) {
                                    boolean ldapLocked = ldapLockedV;
                                    if (ldapLocked != userAccount.locked()) {
                                        LOGGER.info("For userId={}, ldapId={}, setting locked={}", userAccount.id(), ldapUserId, ldapLocked);
                                        if (ldapLocked) {
                                            aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                                        }
                                        userAccount = userAccount.withLocked(ldapLocked);
                                        shouldUpdateInDb = true;
                                    }
                                } else {
                                    LOGGER.warn("For userId={}, ldapId={}, got empty ldap locked", userAccount.id(), ldapUserId);
                                }
                            }

                            if (StringUtils.hasLength(aaaProperties.ldap().attributeNames().enabled())) {
                                var ldapEnabledV = ldapEntry.enabled();
                                if (ldapEnabledV != null) {
                                    boolean ldapEnabled = ldapEnabledV;
                                    if (ldapEnabled != userAccount.enabled()) {
                                        LOGGER.info("For userId={}, ldapId={}, setting enabled={}", userAccount.id(), ldapUserId, ldapEnabled);
                                        if (!ldapEnabled) {
                                            aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                                        }
                                        userAccount = userAccount.withEnabled(ldapEnabled);
                                        shouldUpdateInDb = true;
                                    }
                                } else {
                                    LOGGER.warn("For userId={}, ldapId={}, got empty ldap enabled", userAccount.id(), ldapUserId);
                                }
                            }

                            if (shouldUpdateInDb) {
                                userAccount = userAccount.withSyncLdapTime(currTime);
                                LOGGER.info("Updating userId={}, ldapId={}", userAccount.id(), ldapUserId);
                                userAccountRepository.save(userAccount);
                            } else {
                                toUpdateSyncLdapTime.add(ldapUserId);
                            }
                        } else { // add the user to insert list
                            LOGGER.info("User with ldapId = {} does not exist in database, adding him to insert list", ldapUserId);
                            toInsert.add(ldapEntry);
                        }
                    } else {
                        LOGGER.warn("Got empty ldap userId");
                    }
                } catch (Exception e) {
                    LOGGER.error(e.getMessage(), e);
                }
            }

            LOGGER.info("Inserting {} users to database", toInsert.size());
            var convertedToInsert = toInsert.stream().map(ldapEntry -> {
                Set<String> rawRoles = ldapEntry.roles();
                var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().ldap(), rawRoles);
                boolean locked = ldapEntry.locked() == null ? false : ldapEntry.locked();
                boolean enabled = ldapEntry.enabled() == null ? true : ldapEntry.enabled();
                return UserAccountConverter.buildUserAccountEntityForLdapInsert(
                        ldapEntry.username(),
                        ldapEntry.id(),
                        mappedRoles,
                        ldapEntry.email(),
                        locked,
                        enabled,
                        currTime
                );
            }).toList();
            conflictService.process(convertedToInsert, this);

            if (!toUpdateSyncLdapTime.isEmpty()) {
                LOGGER.info("Updating ldap sync time for {} untoucned users", toUpdateSyncLdapTime.size());
                userAccountRepository.updateSyncLdapTime(toUpdateSyncLdapTime, currTime);
            }
        });
    }

    @Override
    public void saveUser(UserAccount userAccount) {
        userAccountRepository.save(userAccount);
    }


    @Override
    public void saveUsers(Collection<UserAccount> users) {
        userAccountRepository.saveAll(users);
    }

    @Override
    public void removeUser(UserAccount userAccount) {
        userAccountRepository.deleteById(userAccount.id());
    }
}

class MappingConsumingCallbackHandler<T> implements NameClassPairCallbackHandler {

    private final AttributesMapper<T> mapper;

    private final Consumer<List<T>> consumer;

    private final List<T> list = new ArrayList<>();

    private final int batchSize;

    /**
     * Constructs a new instance around the specified {@link AttributesMapper}.
     * @param mapper the target mapper.
     */
    public MappingConsumingCallbackHandler(AttributesMapper<T> mapper, Consumer<List<T>> consumer, int batchSize) {
        this.mapper = mapper;
        this.consumer = consumer;
        this.batchSize = batchSize;
    }

    /**
     * Cast the NameClassPair to a SearchResult and pass its attributes to the
     * {@link AttributesMapper}.
     * @param nameClassPair a <code> SearchResult</code> instance.
     * @return the Object returned from the mapper.
     */
    public T getObjectFromNameClassPairInternal(NameClassPair nameClassPair) {
        if (!(nameClassPair instanceof SearchResult)) {
            throw new IllegalArgumentException("Parameter must be an instance of SearchResult");
        }

        SearchResult searchResult = (SearchResult) nameClassPair;
        Attributes attributes = searchResult.getAttributes();
        try {
            return this.mapper.mapFromAttributes(attributes);
        }
        catch (javax.naming.NamingException ex) {
            throw LdapUtils.convertLdapException(ex);
        }
    }

    @Override
    public final void handleNameClassPair(NameClassPair nameClassPair) throws NamingException {
        this.list.add(getObjectFromNameClassPairInternal(nameClassPair));
        if (list.size() == batchSize) {
            this.consumer.accept(list);
            list.clear();
        }
    }

    public void processLeftovers() {
        if (!list.isEmpty()) {
            this.consumer.accept(list);
            list.clear();
        }
    }
}
