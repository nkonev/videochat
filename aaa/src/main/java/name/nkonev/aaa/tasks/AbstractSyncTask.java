package name.nkonev.aaa.tasks;

import jakarta.annotation.PostConstruct;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.config.properties.RoleMapEntry;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.dto.ExternalSyncEntity;
import name.nkonev.aaa.dto.ForceKillSessionsReasonType;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.security.AaaUserDetailsService;
import name.nkonev.aaa.security.RoleMapper;
import name.nkonev.aaa.services.ConflictResolvingActions;
import name.nkonev.aaa.services.ConflictService;
import name.nkonev.aaa.services.EventService;
import name.nkonev.aaa.utils.Pair;
import org.slf4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.stream.Collectors;

import static name.nkonev.aaa.dto.UserRole.ROLE_USER;
import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

public abstract class AbstractSyncTask<T extends ExternalSyncEntity, TIR extends ExternalSyncEntity> implements ConflictResolvingActions {

    @Autowired
    protected EventService eventService;

    @Autowired
    protected UserAccountRepository userAccountRepository;

    @Autowired
    private ConflictService conflictService;

    @Autowired
    protected TransactionTemplate transactionTemplate;

    @Autowired
    protected AaaUserDetailsService aaaUserDetailsService;

    @PostConstruct
    public void validate() {
        if (!getEnabled()) {
            return;
        }

        if (getConflictResolvingStrategy() == null) {
            throw new IllegalStateException("Conflict resolving strategy is not set");
        }
        getLogger().info("Configured with conflict resolving strategy: {}", getConflictResolvingStrategy());
    }

    public void scheduledTask() {
        try {
            this.doWork();
        } catch (Exception e) {
            getLogger().error("Unexpected exception during doWork()", e);
        }
    }

    protected abstract boolean getEnabled();

    protected abstract Logger getLogger();

    public void doWork() {
        var currTime = getNowUTC();
        doConcreteWork(currTime);
    }

    // should invoke processUpsertBatch() several times and processDeleted() one time
    protected abstract void doConcreteWork(LocalDateTime currTime);

    protected void sendEvents(List<EventWrapper<?>> events) {
        final int maxEventsBeforeCanThrottle = getMaxEventsBeforeCanThrottle();

        int counter = 0;
        for (EventWrapper<?> event : events) {
            if (counter >= maxEventsBeforeCanThrottle) {
                event = event.withCanThrottle(true);
            }
            eventService.sendProfileEvent(event);

            counter++;
        }
        events.clear();
    }

    @Override
    public void insertUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer) {
        var saved = userAccountRepository.save(userAccount);
        eventsContainer.add(eventService.convertProfileCreated(saved));
    }

    @Override
    public void updateUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer) {
        eventsContainer.add(eventService.convertProfileUpdated(userAccount));
        userAccountRepository.save(userAccount);
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
        eventsContainer.add(eventService.convertProfileDeleted(userAccount.id()));
        userAccountRepository.deleteById(userAccount.id());
    }

    // processing resulting into (new users) inserts and (new users) updates
    protected void processUpsertBatch(List<T> entries, LocalDateTime currTime) {
        List<EventWrapper<?>> eventsContainer = new ArrayList<>();
        transactionTemplate.executeWithoutResult(s -> {
            Map<String, T> byExtIdId = new HashMap<>();
            for (var extEntry : entries) {
                var extUserId = extEntry.getId();
                if (StringUtils.hasLength(extUserId)) {
                    byExtIdId.put(extUserId, extEntry);
                }
            }
            var dbChunk = findByExtId(byExtIdId.keySet());

            var toInsert = new ArrayList<T>();
            var toUpdateSetExtSyncTime = new HashSet<String>();
            for (var entry : byExtIdId.entrySet()) {
                try {
                    var extUserId = entry.getKey();
                    var extEntry = entry.getValue();
                    getLogger().debug("Examining user with {}Id={}", getName(), extUserId);

                    if (StringUtils.hasLength(extUserId)) {
                        var o = findByExtUserId(dbChunk, extUserId);
                        if (o.isPresent()) { // update the existing user
                            getLogger().debug("User with {}Id={} is existing in database, deciding to update him or not", getName(), extUserId);
                            var userAccount = o.get();

                            var p = applyUpdateToUserAccount(extEntry, userAccount);
                            userAccount = p.a();
                            boolean shouldUpdateInDb = p.b();

                            if (shouldUpdateInDb) {
                                userAccount = setSyncTime(userAccount, currTime);
                                getLogger().info("Updating userId={}, {}Id={}", userAccount.id(), getName(), extUserId);

                                conflictService.process(getRenamingPrefix(), getConflictResolvingStrategy(), ConflictService.PotentiallyConflictingAction.UPDATE, userAccount, this, eventsContainer);
                            } else {
                                toUpdateSetExtSyncTime.add(extUserId);
                            }
                        } else { // add the user to insert list
                            getLogger().info("User with {}Id = {} does not exist in database, adding him to insert list", getName(), extUserId);
                            toInsert.add(extEntry);
                        }
                    } else {
                        getLogger().warn("Got empty {}userId", getName());
                    }
                } catch (Exception e) {
                    getLogger().error(e.getMessage(), e);
                }
            }

            getLogger().info("Inserting {} users to database", toInsert.size());
            var convertedToInsert = toInsert.stream().map(entry -> this.prepareUserAccountForInsert(entry, currTime)).toList();
            conflictService.process(getRenamingPrefix(), getConflictResolvingStrategy(), ConflictService.PotentiallyConflictingAction.INSERT, convertedToInsert, this, eventsContainer);

            if (!toUpdateSetExtSyncTime.isEmpty()) {
                getLogger().info("Updating {} sync time for {} untoucned users", getName(), toUpdateSetExtSyncTime.size());
                batchSetSyncTime(toUpdateSetExtSyncTime, currTime);
            }
        });

        sendEvents(eventsContainer);
    }

    protected void processDeleted(int size, LocalDateTime currTime) {
        var shouldContinue = new AtomicBoolean(true);
        for (int offset = 0; shouldContinue.get(); offset += size) {
            List<EventWrapper<?>> eventsContainer = new ArrayList<>();
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                var toDelete = findExtIdsElderThan(size, theOffset, currTime);
                toDelete.forEach(userIdToDelete -> eventsContainer.add(eventService.convertProfileDeleted(userIdToDelete)));
                userAccountRepository.deleteAllById(toDelete);
                getLogger().info("Deleted users with ids {} from database which were removed from {}", toDelete, getName());
                shouldContinue.set(toDelete.size() == size);
            });
            sendEvents(eventsContainer);
        }
    }

    protected void processAddingRoleToUsers(List<TIR> extUsers, String extRole, List<EventWrapper<?>> eventsContainer, LocalDateTime currTime) {
        if (extUsers.isEmpty()) {
            return;
        }
        var extIds = extUsers.stream().map(TIR::getId).toList();
        var dbUsers = findByExtIdInOrderById(extIds);

        var mappedToDbRole = RoleMapper.map(getRoleMappings(), extRole);

        var toUpdateTimeInDb = new HashSet<String>();
        var toUpdateInDb = new ArrayList<UserAccount>();
        for (var dbUser : dbUsers) {
            var dbUserRoles = Arrays.stream(dbUser.roles()).collect(Collectors.toCollection(HashSet::new));
            var extUserOptional = getExtUserOptional(dbUser, extUsers);
            extUserOptional.ifPresent(extUser -> {
                if (!dbUserRoles.contains(mappedToDbRole)) {
                    getLogger().info("Adding role {} to user id = {}, {}Id = {}", mappedToDbRole, dbUser.id(), getName(), getExtId(dbUser));
                    aaaUserDetailsService.killSessions(dbUser.id(), ForceKillSessionsReasonType.user_roles_changed);
                    dbUserRoles.add(mappedToDbRole);
                    var changedDbUser = setSyncExtRolesTime(dbUser
                            .withRoles(dbUserRoles.toArray(new UserRole[0])), currTime);
                    toUpdateInDb.add(changedDbUser);
                } else {
                    toUpdateTimeInDb.add(getExtId(dbUser));
                }
            }); // if not existed - it is handled in the different place
        }

        if (!toUpdateInDb.isEmpty()) {
            updateUsers(toUpdateInDb, eventsContainer);
        }

        if (!toUpdateTimeInDb.isEmpty()) {
            updateSyncExtRolesTime(toUpdateTimeInDb, currTime);
        }
    }

    protected void processRemovingRolesFromUsers(int batchSize, LocalDateTime currTime) {
        var shouldContinue2 = new AtomicBoolean(true);
        for (var offset = 0; shouldContinue2.get(); offset += batchSize) {
            List<EventWrapper<?>> eventsContainer = new ArrayList<>();
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                var toMakeWithoutAdminRole = findExtIdsRolesElderThan(batchSize, theOffset, currTime); // process almost all users, because typically it's very low amount of admins
                shouldContinue2.set(toMakeWithoutAdminRole.size() == batchSize);

                var toSaveToDb = toMakeWithoutAdminRole.stream()
                        .map(u -> {
                            if (Arrays.stream(u.roles()).collect(Collectors.toSet()).contains(UserRole.ROLE_ADMIN)) {
                                getLogger().info("Removing role {} from user id = {}, login = {}, {}Id = {}", UserRole.ROLE_ADMIN, u.id(), u.username(), getName(), getExtId(u));
                                aaaUserDetailsService.killSessions(u.id(), ForceKillSessionsReasonType.user_roles_changed);
                                eventsContainer.add(eventService.convertProfileUpdated(u));
                                return setSyncExtRolesTime(u
                                        .withRoles(new UserRole[]{ROLE_USER}), currTime);
                            } else {
                                return setSyncExtRolesTime(u, currTime);
                            }
                        })
                        .toList();
                userAccountRepository.saveAll(toSaveToDb);
            });
            sendEvents(eventsContainer);
        }
    }

    protected abstract UserAccount prepareUserAccountForInsert(T t, LocalDateTime currTime);

    protected abstract Pair<UserAccount, Boolean> applyUpdateToUserAccount(T entity, UserAccount userAccount);

    protected abstract String getName();

    protected abstract List<UserAccount> findByExtId(Collection<String> extIds);

    protected abstract UserAccount setSyncTime(UserAccount userAccount, LocalDateTime currTime);

    protected abstract void batchSetSyncTime(Set<String> toUpdateSetExtSyncTime, LocalDateTime currTime);

    protected abstract Optional<UserAccount> findByExtUserId(List<UserAccount> dbChunk, String extUserId);

    protected abstract List<Long> findExtIdsElderThan(int limit, int theOffset, LocalDateTime currTime);

    protected abstract ConflictResolveStrategy getConflictResolvingStrategy();

    protected abstract String getRenamingPrefix();

    protected abstract String getNecessaryAdminRole();

    protected abstract List<RoleMapEntry> getRoleMappings();

    protected abstract List<UserAccount> findByExtIdInOrderById(Collection<String> extIds);

    protected abstract Optional<TIR> getExtUserOptional(UserAccount dbUser, List<TIR> extUsers);

    protected abstract String getExtId(UserAccount u);

    protected abstract void updateSyncExtRolesTime(Set<String> toUpdateTimeInDb, LocalDateTime currTime);

    protected abstract UserAccount setSyncExtRolesTime(UserAccount userAccount, LocalDateTime currTime);

    protected abstract List<UserAccount> findExtIdsRolesElderThan(int limit, int theOffset, LocalDateTime currTime);

    protected abstract int getMaxEventsBeforeCanThrottle();
}
