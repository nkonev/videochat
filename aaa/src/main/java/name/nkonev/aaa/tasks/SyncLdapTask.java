package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.config.properties.RoleMapEntry;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.ldap.LdapEntity;
import name.nkonev.aaa.entity.ldap.LdapUserInRoleEntity;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.services.LockService;
import name.nkonev.aaa.services.tasks.LdapMappingConsumingCallbackHandler;
import name.nkonev.aaa.services.tasks.LdapSyncRolesService;
import name.nkonev.aaa.utils.Pair;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ldap.core.*;
import org.springframework.ldap.filter.WhitespaceWildcardsFilter;
import org.springframework.ldap.query.LdapQueryBuilder;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import javax.naming.directory.Attributes;
import javax.naming.directory.DirContext;
import javax.naming.directory.SearchControls;

import static name.nkonev.aaa.Constants.LDAP_CONFLICT_PREFIX;
import static name.nkonev.aaa.utils.RoleUtils.DEFAULT_ROLE;

@Service
public class SyncLdapTask extends AbstractSyncTask<LdapEntity, LdapUserInRoleEntity> {
    private final AaaProperties aaaProperties;

    private final LdapOperations ldapOperations;

    private final AaaUserDetailsService aaaUserDetailsService;

    private final LdapSyncRolesService ldapSyncRolesService;

    private final LockService lockService;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncLdapTask.class);

    private static final String LOCK_NAME = "syncLdapTask";

    public SyncLdapTask(AaaProperties aaaProperties, LdapOperations ldapOperations, AaaUserDetailsService aaaUserDetailsService, LdapSyncRolesService ldapSyncRolesService, LockService lockService) {
        this.aaaProperties = aaaProperties;
        this.ldapOperations = ldapOperations;
        this.aaaUserDetailsService = aaaUserDetailsService;
        this.ldapSyncRolesService = ldapSyncRolesService;
        this.lockService = lockService;
    }

    @Scheduled(cron = "${custom.schedulers.sync-ldap.cron}")
    public void scheduledTask() {
        if (!getEnabled()) {
            return;
        }

        try (var l = lockService.lock(LOCK_NAME, aaaProperties.schedulers().syncLdap().expiration())) {
            if (l.isWasSet()) {
                super.scheduledTask();
            }
        }
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
    protected void doConcreteWork(LocalDateTime currTime) {
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
        LdapMappingConsumingCallbackHandler<LdapEntity> handler = new LdapMappingConsumingCallbackHandler<>(mapper, entries -> this.processUpsertBatch(entries, currTime), batchSize);
        ldapOperations.search(se, handler);
        handler.processLeftovers();

        if (aaaProperties.schedulers().syncLdap().syncRoles()) {
            LOGGER.info("Syncing roles from LDAP");
            processRoles(batchSize, currTime);
        }

        LOGGER.info("Deleting entries from database which were removed from LDAP");
        processDeleted(batchSize, currTime);

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
                        aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_disabled);
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
    protected UserAccount prepareUserAccountForInsert(LdapEntity ldapEntry, LocalDateTime currTime) {
        var mappedRoles = Set.of(DEFAULT_ROLE);
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
    protected UserAccount setSyncTime(UserAccount userAccount, LocalDateTime currTime) {
        return userAccount.withSyncLdapTime(currTime);
    }

    @Override
    protected void batchSetSyncTime(Set<String> toUpdateSetExtSyncTime, LocalDateTime currTime) {
        userAccountRepository.updateSyncLdapTime(toUpdateSetExtSyncTime, currTime);
    }

    @Override
    protected Optional<UserAccount> findByExtUserId(List<UserAccount> dbChunk, String extUserId) {
        return dbChunk.stream().filter(ua -> ua.ldapId().equals(extUserId)).findFirst();
    }

    @Override
    protected List<Long> findExtIdsElderThan(int limit, int theOffset, LocalDateTime currTime) {
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

    @Override
    protected String getNecessaryAdminRole() {
        return ldapSyncRolesService.getNecessaryAdminRole();
    }

    @Override
    protected List<RoleMapEntry> getRoleMappings() {
        return aaaProperties.roleMappings().ldap();
    }

    @Override
    protected Optional<LdapUserInRoleEntity> getExtUserOptional(UserAccount dbUser, List<LdapUserInRoleEntity> extUsers) {
        return extUsers.stream().filter(du -> du.id().equals(dbUser.ldapId())).findFirst();
    }

    @Override
    protected String getExtId(UserAccount u) {
        return u.ldapId();
    }

    @Override
    protected UserAccount setSyncExtRolesTime(UserAccount userAccount, LocalDateTime currTime) {
        return userAccount.withSyncLdapRolesTime(currTime);
    }

    @Override
    protected List<UserAccount> findExtIdsRolesElderThan(int limit, int theOffset, LocalDateTime currTime) {
        return userAccountRepository.findByLdapIdRolesElderThan(currTime, limit, theOffset);
    }

    @Override
    protected void updateSyncExtRolesTime(Set<String> toUpdateTimeInDb, LocalDateTime currTime) {
        userAccountRepository.updateSyncLdapRolesTime(toUpdateTimeInDb, currTime);

    }

    @Override
    protected List<UserAccount> findByExtIdInOrderById(Collection<String> extIds) {
        return userAccountRepository.findByLdapIdInOrderById(extIds);
    }

    private void processRoles(int batchSize, LocalDateTime currTime) {
        var extAdminRole = getNecessaryAdminRole();
        ldapSyncRolesService.processRoles(batchSize, extAdminRole, batch -> processRolesBatch(extAdminRole, batch, currTime));

        // remove admin role
        processRemovingRolesFromUsers(batchSize, currTime);
    }

    private void processRolesBatch(String extAdminRole, List<LdapUserInRoleEntity> extUsersInRole, LocalDateTime currTime) {
        List<EventWrapper<?>> eventsContainer = new ArrayList<>();
        transactionTemplate.executeWithoutResult(s -> {
            processAddingRoleToUsers(extUsersInRole, extAdminRole, eventsContainer, currTime);
        });
        sendEvents(eventsContainer);
    }

    @Override
    protected int getMaxEventsBeforeCanThrottle() {
        return aaaProperties.schedulers().syncLdap().maxEventsBeforeCanThrottle();
    }
}

