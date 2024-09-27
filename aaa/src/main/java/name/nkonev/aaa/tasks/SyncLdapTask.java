package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.ldap.LdapEntity;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.security.RoleMapper;
import name.nkonev.aaa.utils.Pair;
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
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.function.Consumer;
import java.util.stream.Collectors;
import javax.naming.NameClassPair;
import javax.naming.NamingException;
import javax.naming.directory.Attributes;
import javax.naming.directory.DirContext;
import javax.naming.directory.SearchControls;
import javax.naming.directory.SearchResult;

import static name.nkonev.aaa.Constants.LDAP_CONFLICT_PREFIX;

@Service
public class SyncLdapTask extends AbstractSyncTask<LdapEntity> {
    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private LdapOperations ldapOperations;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncLdapTask.class);

    @Scheduled(cron = "${custom.schedulers.sync-ldap.cron}")
    @SchedulerLock(name = "syncLdapTask")
    public void scheduledTask() {
        super.scheduledTask();
    }

    @Override
    protected boolean getEnabled() {
        return aaaProperties.schedulers().syncLdap().enabled();
    }

    @Override
    protected Logger getLogger() {
        return LOGGER;
    }

    @Override
    protected void doConcreteWork() {
        final var batchSize = aaaProperties.schedulers().syncLdap().batchSize();
        LOGGER.info("Sync ldap task start, batchSize={}", batchSize);

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
        MappingConsumingCallbackHandler<LdapEntity> handler = new MappingConsumingCallbackHandler<>(mapper, this::processUpsertBatch, batchSize);
        ldapOperations.search(se, handler);
        handler.processLeftovers();

        LOGGER.info("Deleting entries from database which were removed from LDAP");
        processDeleted(batchSize);

        LOGGER.info("Sync ldap task finish");
    }

    @Override
    protected Pair<UserAccount, Boolean> applyUpdateToUserAccount(LdapEntity ldapEntry, UserAccount userAccount) {
        boolean shouldUpdateInDb = false;
        var ldapUserId = ldapEntry.getId();
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
                LOGGER.warn("For userId={}, ldapId={}, got empty ldap's enabled", userAccount.id(), ldapUserId);
            }
        }

        return new Pair<>(userAccount, shouldUpdateInDb);
    }

    @Override
    protected UserAccount prepareUserAccountForInsert(LdapEntity ldapEntry) {
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
    }

    @Override
    protected String getName() {
        return "ldap";
    }

    @Override
    protected List<UserAccount> findByExtId(Collection<String> extIds) {
        return userAccountRepository.findByLdapIdInOrderById(extIds);
    }

    @Override
    protected UserAccount setSyncTime(UserAccount userAccount) {
        return userAccount.withSyncLdapTime(currTime);
    }

    @Override
    protected void batchSetSyncTime(Set<String> toUpdateSetExtSyncTime) {
        userAccountRepository.updateSyncLdapTime(toUpdateSetExtSyncTime, currTime);
    }

    @Override
    protected Optional<UserAccount> findByExtUserId(List<UserAccount> dbChunk, String extUserId) {
        return dbChunk.stream().filter(ua -> ua.ldapId().equals(extUserId)).findFirst();
    }

    @Override
    protected List<Long> findExtIdsElderThan(int limit, int theOffset) {
        return userAccountRepository.findByLdapIdElderThan(currTime, limit, theOffset);
    }

    @Override
    protected ConflictResolveStrategy getConflictResolvingStrategy() {
        return aaaProperties.ldap().resolveConflictsStrategy();
    }

    @Override
    protected String getRenamingPrefix() {
        return LDAP_CONFLICT_PREFIX;
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
