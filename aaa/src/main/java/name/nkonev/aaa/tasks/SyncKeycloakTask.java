package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.config.properties.RoleMapEntry;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.rest.KeycloakUserEntity;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.rest.KeycloakUserInRoleEntity;
import name.nkonev.aaa.services.tasks.KeycloakClient;
import name.nkonev.aaa.utils.Pair;
import net.javacrumbs.shedlock.spring.annotation.SchedulerLock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;

import static name.nkonev.aaa.Constants.KEYCLOAK_CONFLICT_PREFIX;
import static name.nkonev.aaa.security.OAuth2Providers.KEYCLOAK;
import static name.nkonev.aaa.utils.RoleUtils.DEFAULT_ROLE;


@Service
public class SyncKeycloakTask extends AbstractSyncTask<KeycloakUserEntity, KeycloakUserInRoleEntity> {

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private ObjectProvider<KeycloakClient> keycloakClientProvider;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncKeycloakTask.class);

    private KeycloakClient keycloakClient;

    @Scheduled(cron = "${custom.schedulers.sync-keycloak.cron}")
    @SchedulerLock(name = "syncKeycloakTask")
    public void scheduledTask() {
        super.scheduledTask();
    }

    @Override
    protected boolean getEnabled() {
        return aaaProperties.schedulers().syncKeycloak().enabled();
    }

    @Override
    protected Logger getLogger() {
        return LOGGER;
    }

    @Override
    protected void doConcreteWork() {
        var keycloakClient = keycloakClientProvider.getIfAvailable();
        if (keycloakClient == null) {
            LOGGER.error("Keycloak client is not configured, you must to add its OAuth provider and registration");
            return;
        }
        this.keycloakClient = keycloakClient;

        final var batchSize = aaaProperties.schedulers().syncKeycloak().batchSize();
        LOGGER.info("Sync Keycloak task start, batchSize={}", batchSize);

        LOGGER.info("Upserting entries from Keycloak");
        var shouldContinue = true;
        for (int offset = 0; shouldContinue; offset += batchSize) {
            var users = this.keycloakClient.getUsers(batchSize, offset);
            processUpsertBatch(users);
            if (users.size() < batchSize) {
                shouldContinue = false;
            }
        }

        if (aaaProperties.schedulers().syncKeycloak().syncRoles()) {
            LOGGER.info("Syncing roles from Keycloak");
            processRoles(batchSize);
        }

        LOGGER.info("Deleting entries from database which were removed from Keycloak");
        processDeleted(batchSize);

        LOGGER.info("Sync Keycloak task finish");
    }

    @Override
    protected Pair<UserAccount, Boolean> applyUpdateToUserAccount(KeycloakUserEntity keycloakEntry, UserAccount userAccount) {
        boolean shouldUpdateInDb = false;
        var keycloakUserId = keycloakEntry.getId();
        var keycloakUsername = keycloakEntry.username();
        if (StringUtils.hasLength(keycloakUsername)) {
            if (!keycloakUsername.equals(userAccount.username())) {
                LOGGER.info("For userId={}, keycloakId={}, setting username={}", userAccount.id(), keycloakUserId, keycloakUsername);
                userAccount = userAccount.withUsername(keycloakUsername);
                shouldUpdateInDb = true;
            }
        } else {
            LOGGER.warn("For userId={}, keycloakId={}, got empty keycloak's username", userAccount.id(), keycloakUserId);
        }

        var keycloakEmail = keycloakEntry.email();
        if (!Objects.equals(keycloakEmail, userAccount.email())) {
            LOGGER.info("For userId={}, keycloakId={}, setting email={}", userAccount.id(), keycloakUserId, keycloakEmail);
            userAccount = userAccount.withEmail(keycloakEmail);
            shouldUpdateInDb = true;
        }

        var keycloakEnabledV = keycloakEntry.enabled();
        if (keycloakEnabledV != null) {
            boolean keycloakEnabled = keycloakEnabledV;
            if (keycloakEnabled != userAccount.enabled()) {
                LOGGER.info("For userId={}, keycloakId={}, setting enabled={}", userAccount.id(), keycloakUserId, keycloakEnabled);
                if (!keycloakEnabled) {
                    aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                }
                userAccount = userAccount.withEnabled(keycloakEnabled);
                shouldUpdateInDb = true;
            }
        } else {
            LOGGER.warn("For userId={}, keycloakId={}, got empty keycloak's enabled", userAccount.id(), keycloakUserId);
        }

        if (aaaProperties.schedulers().syncKeycloak().syncEmailVerified()) {
            var keycloakEmailVerifiedV = keycloakEntry.emailVerified();
            if (keycloakEmailVerifiedV != null) {
                boolean keycloakEmailVerified = keycloakEmailVerifiedV;
                if (keycloakEmailVerified != userAccount.confirmed()) {
                    LOGGER.info("For userId={}, keycloakId={}, setting confirmed={}", userAccount.id(), keycloakUserId, keycloakEmailVerified);
                    if (!keycloakEmailVerified) {
                        aaaUserDetailsService.killSessions(userAccount.id(), ForceKillSessionsReasonType.user_locked);
                    }
                    userAccount = userAccount.withConfirmed(keycloakEmailVerified);
                    shouldUpdateInDb = true;
                }
            } else {
                LOGGER.warn("For userId={}, keycloakId={}, got empty keycloak's email verified", userAccount.id(), keycloakUserId);
            }
        }

        return new Pair<>(userAccount, shouldUpdateInDb);
    }

    @Override
    protected UserAccount prepareUserAccountForInsert(KeycloakUserEntity keycloakUserEntity) {
        var roles = Set.of(DEFAULT_ROLE);
        boolean locked = false;
        boolean enabled = keycloakUserEntity.enabled();
        return UserAccountConverter.buildUserAccountEntityForKeycloakInsert(
                keycloakUserEntity.id(),
                keycloakUserEntity.username(),
                null,
                roles,
                keycloakUserEntity.email(),
                locked,
                enabled,
                currTime
        );
    }

    @Override
    protected String getName() {
        return KEYCLOAK;
    }

    @Override
    protected List<UserAccount> findByExtId(Collection<String> extIds) {
        return userAccountRepository.findByKeycloakIdInOrderById(extIds);
    }

    @Override
    protected UserAccount setSyncTime(UserAccount userAccount) {
        return userAccount.withSyncKeycloakTime(currTime);
    }

    @Override
    protected void batchSetSyncTime(Set<String> toUpdateSetExtSyncTime) {
        userAccountRepository.updateSyncKeycloakTime(toUpdateSetExtSyncTime, currTime);
    }

    @Override
    protected Optional<UserAccount> findByExtUserId(List<UserAccount> dbChunk, String extUserId) {
        return dbChunk.stream().filter(ua -> ua.keycloakId().equals(extUserId)).findFirst();
    }

    @Override
    protected List<Long> findExtIdsElderThan(int limit, int theOffset) {
        return userAccountRepository.findByKeycloakIdElderThan(currTime, limit, theOffset);
    }

    @Override
    protected ConflictResolveStrategy getConflictResolvingStrategy() {
        return aaaProperties.keycloak().resolveConflictsStrategy();
    }

    @Override
    protected String getRenamingPrefix() {
        return KEYCLOAK_CONFLICT_PREFIX;
    }

    @Override
    protected String getNecessaryAdminRole() {
        var list = aaaProperties.roleMappings().keycloak().stream()
                .filter(roleMapEntry -> UserRole.ROLE_ADMIN.name().equals(roleMapEntry.our()))
                .map(RoleMapEntry::their)
                .toList();
        if (list.isEmpty()) {
            throw new IllegalStateException("Admin role not found in mapping");
        }
        return list.getFirst();
    }

    @Override
    protected List<RoleMapEntry> getRoleMappings() {
        return aaaProperties.roleMappings().keycloak();
    }

    @Override
    protected List<UserAccount> findByExtIdInOrderById(Collection<String> keycloakIds) {
        return userAccountRepository.findByKeycloakIdInOrderById(keycloakIds);
    }

    @Override
    protected Optional<KeycloakUserInRoleEntity> getExtUserOptional(UserAccount dbUser, List<KeycloakUserInRoleEntity> keycloakUsers) {
        return keycloakUsers.stream().filter(du -> du.id().equals(dbUser.keycloakId())).findFirst();
    }

    @Override
    protected String getExtId(UserAccount dbUser) {
        return dbUser.keycloakId();
    }

    @Override
    protected void updateSyncExtRolesTime(Set<String> toUpdateTimeInDb) {
        userAccountRepository.updateSyncKeycloakRolesTime(toUpdateTimeInDb, currTime);
    }

    @Override
    protected UserAccount setSyncExtRolesTime(UserAccount userAccount) {
        return userAccount.withSyncKeycloakRolesTime(currTime);
    }

    @Override
    protected List<UserAccount> findExtIdsRolesElderThan(int limit, int theOffset) {
        return userAccountRepository.findByKeycloakIdRolesElderThan(currTime, limit, theOffset);
    }

    private void processRoles(int batchSize) {
        var extAdminRole = getNecessaryAdminRole();
        var shouldContinue = new AtomicBoolean(true);
        for (int offset = 0; shouldContinue.get(); offset += batchSize) {
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                List<KeycloakUserInRoleEntity> extUsersInRole = this.keycloakClient.getUsersInRole(extAdminRole, batchSize, theOffset);
                processAddingRoleToUsers(extUsersInRole, extAdminRole);
                shouldContinue.set(extUsersInRole.size() == batchSize);
            });
            sendEvents();
        }

        // remove admin role
        processRemovingRolesFromUsers(batchSize);
    }
}

