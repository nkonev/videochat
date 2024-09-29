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
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.security.RoleMapper;
import name.nkonev.aaa.utils.Pair;
import net.javacrumbs.shedlock.spring.annotation.SchedulerLock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.stream.Collectors;

import static name.nkonev.aaa.Constants.KEYCLOAK_CONFLICT_PREFIX;
import static name.nkonev.aaa.dto.UserRole.ROLE_USER;
import static name.nkonev.aaa.security.OAuth2Providers.KEYCLOAK;
import static name.nkonev.aaa.utils.RoleUtils.DEFAULT_ROLE;


@Service
public class SyncKeycloakTask extends AbstractSyncTask<KeycloakUserEntity> {

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private ObjectProvider<KeycloakClient> keycloakClientProvider;

    private static final Logger LOGGER = LoggerFactory.getLogger(SyncKeycloakTask.class);
    @Autowired
    private TransactionTemplate transactionTemplate;

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

        final var batchSize = aaaProperties.schedulers().syncKeycloak().batchSize();
        LOGGER.info("Sync Keycloak task start, batchSize={}", batchSize);

        LOGGER.info("Upserting entries from Keycloak");
        var shouldContinue = true;
        for (int offset = 0; shouldContinue; offset += batchSize) {
            var users = keycloakClient.getUsers(batchSize, offset);
            processUpsertBatch(users);
            if (users.size() < batchSize) {
                shouldContinue = false;
            }
        }

        LOGGER.info("Syncing roles from Keycloak");
        processRoles(keycloakClient, batchSize);

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

    private void processRoles(KeycloakClient keycloakClient, int batchSize) {
        var keycloakAdminRole = getNecessaryKeycloakAdminRole();
        var shouldContinue = new AtomicBoolean(true);
        for (int offset = 0; shouldContinue.get(); offset += batchSize) {
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                List<KeycloakUserInRoleEntity> keycloakUsers = keycloakClient.getUsersInRole(keycloakAdminRole, batchSize, theOffset);
                processAddingRoleToUsers(keycloakUsers, keycloakAdminRole);
                shouldContinue.set(keycloakUsers.size() == batchSize);
            });
            sendEvents();
        }

        // remove admin role
        processRemovingRolesFromUsers(batchSize);
    }

    private void processRemovingRolesFromUsers(int batchSize) {
        var shouldContinue2 = new AtomicBoolean(true);
        for (var offset = 0; shouldContinue2.get(); offset += batchSize) {
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                var toMakeWithoutAdminRole = userAccountRepository.findByKeycloakIdRolesElderThan(currTime, batchSize, theOffset); // process almost all users, because typically it's very low amount of admins
                shouldContinue2.set(toMakeWithoutAdminRole.size() == batchSize);

                var toSaveToDb = toMakeWithoutAdminRole.stream()
                        .map(u -> {
                            if (Arrays.stream(u.roles()).collect(Collectors.toSet()).contains(UserRole.ROLE_ADMIN)) {
                                LOGGER.info("Removing role {} from user id = {}, login = {}, keycloakId = {}", UserRole.ROLE_ADMIN, u.id(), u.username(), u.keycloakId());
                                aaaUserDetailsService.killSessions(u.id(), ForceKillSessionsReasonType.user_roles_changed);
                                events.add(eventService.convertProfileUpdated(u));
                                return u
                                        .withRoles(new UserRole[]{ROLE_USER})
                                        .withSyncKeycloakRolesTime(currTime);
                            } else {
                                return u
                                        .withSyncKeycloakRolesTime(currTime);
                            }
                        })
                        .toList();
                userAccountRepository.saveAll(toSaveToDb);
            });
            sendEvents();
        }
    }

    private void processAddingRoleToUsers(List<KeycloakUserInRoleEntity> keycloakUsers, String keycloakRole) {
        if (keycloakUsers.isEmpty()) {
            return;
        }
        var keycloakIds = keycloakUsers.stream().map(KeycloakUserInRoleEntity::getId).toList();
        var dbUsers = userAccountRepository.findByKeycloakIdInOrderById(keycloakIds);

        var mappedToDbRole = RoleMapper.map(aaaProperties.roleMappings().keycloak(), keycloakRole);

        var toUpdateTimeInDb = new HashSet<String>();
        var toUpdateInDb = new ArrayList<UserAccount>();
        for (var dbUser : dbUsers) {
            var dbUserRoles = Arrays.stream(dbUser.roles()).collect(Collectors.toCollection(HashSet::new));
            var keycloakUserOptional = keycloakUsers.stream().filter(du -> du.id().equals(dbUser.keycloakId())).findFirst();
            keycloakUserOptional.ifPresent(keycloakUser -> {
                if (!dbUserRoles.contains(mappedToDbRole)) {
                    LOGGER.info("Adding role {} to user id = {}, keycloakId = {}", mappedToDbRole, dbUser.id(), dbUser.keycloakId());
                    aaaUserDetailsService.killSessions(dbUser.id(), ForceKillSessionsReasonType.user_roles_changed);
                    dbUserRoles.add(mappedToDbRole);
                    var changedDbUser = dbUser
                            .withRoles(dbUserRoles.toArray(new UserRole[0]))
                            .withSyncKeycloakRolesTime(currTime);
                    toUpdateInDb.add(changedDbUser);
                } else {
                    toUpdateTimeInDb.add(dbUser.keycloakId());
                }
            }); // if not existed - it is handled in the different place
        }

        if (!toUpdateInDb.isEmpty()) {
            updateUsers(toUpdateInDb);
        }

        if (!toUpdateTimeInDb.isEmpty()) {
            userAccountRepository.updateSyncKeycloakRolesTime(toUpdateTimeInDb, currTime);
        }
    }

    private String getNecessaryKeycloakAdminRole() {
        return aaaProperties.roleMappings().keycloak().stream()
                .filter(roleMapEntry -> UserRole.ROLE_ADMIN.name().equals(roleMapEntry.our()))
                .map(RoleMapEntry::their)
                .toList().get(0);
    }
}

