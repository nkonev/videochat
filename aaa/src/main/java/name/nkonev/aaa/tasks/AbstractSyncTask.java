package name.nkonev.aaa.tasks;

import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.services.ConflictResolvingActions;
import name.nkonev.aaa.services.ConflictService;
import name.nkonev.aaa.services.EventService;
import name.nkonev.aaa.utils.PageUtils;
import name.nkonev.aaa.utils.Pair;
import org.slf4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

// This service is designed for a single-thread using
public abstract class AbstractSyncTask<T extends ExternalSyncEntity> implements ConflictResolvingActions {

    @Autowired
    protected EventService eventService;

    @Autowired
    protected UserAccountRepository userAccountRepository;

    @Autowired
    private ConflictService conflictService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    protected LocalDateTime currTime;

    protected final List<EventWrapper<?>> events = new ArrayList<>();

    public void scheduledTask() {
        if (!getEnabled()) {
            return;
        }

        try {
            this.doWork();
        } catch (Exception e) {
            getLogger().error("Unexpected exception during doWork()", e);
            sendEvents(); // for the case
        }
    }

    protected abstract boolean getEnabled();

    protected abstract Logger getLogger();

    public void doWork() {
        currTime = getNowUTC();
        doConcreteWork();
        sendEvents(); // for the case
    }

    // should invoke processUpsertBatch() several times and processDeleted() one time
    protected abstract void doConcreteWork();

    protected void sendEvents() {
        for (EventWrapper<?> event : events) {
            eventService.sendProfileEvent(event);
        }
        events.clear();
    }

    @Override
    public void insertUser(UserAccount userAccount) {
        var saved = userAccountRepository.save(userAccount);
        events.add(eventService.convertProfileCreated(saved));
    }

    @Override
    public void updateUser(UserAccount userAccount) {
        events.add(eventService.convertProfileUpdated(userAccount));
        userAccountRepository.save(userAccount);
    }

    @Override
    public void insertUsers(Collection<UserAccount> users) {
        var saved = userAccountRepository.saveAll(users);
        for (UserAccount userAccount : saved) {
            events.add(eventService.convertProfileCreated(userAccount));
        }
    }

    public void updateUsers(Collection<UserAccount> users) {
        for (UserAccount userAccount : users) {
            events.add(eventService.convertProfileUpdated(userAccount));
        }
        userAccountRepository.saveAll(users);
    }

    @Override
    public void removeUser(UserAccount userAccount) {
        events.add(eventService.convertProfileDeleted(userAccount.id()));
        userAccountRepository.deleteById(userAccount.id());
    }

    // processing resulting into (new users) inserts and (new users) updates
    protected void processUpsertBatch(List<T> entries) {
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
                    getLogger().info("Examining user with {}Id={}", getName(), extUserId);

                    if (StringUtils.hasLength(extUserId)) {
                        var o = findByExtUserId(dbChunk, extUserId);
                        if (o.isPresent()) { // update the existing user
                            getLogger().info("User with {}Id={} is existing in database, deciding to update him or not", getName(), extUserId);
                            var userAccount = o.get();

                            var p = applyUpdateToUserAccount(extEntry, userAccount);
                            userAccount = p.a();
                            boolean shouldUpdateInDb = p.b();

                            if (shouldUpdateInDb) {
                                userAccount = setSyncTime(userAccount);
                                getLogger().info("Updating userId={}, {}Id={}", userAccount.id(), getName(), extUserId);
                                updateUser(userAccount);
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
            var convertedToInsert = toInsert.stream().map(this::prepareUserAccountForInsert).toList();
            conflictService.process(getRenamingPrefix(), getConflictResolvingStrategy(), convertedToInsert, this);

            if (!toUpdateSetExtSyncTime.isEmpty()) {
                getLogger().info("Updating {} sync time for {} untoucned users", getName(), toUpdateSetExtSyncTime.size());
                batchSetSyncTime(toUpdateSetExtSyncTime);
            }
        });
        sendEvents();
    }

    protected void processDeleted() {
        var shouldContinue = new AtomicBoolean(true);
        final var size = PageUtils.DEFAULT_SIZE;
        for (int offset = 0; shouldContinue.get(); offset += size) {
            final var theOffset = offset;
            transactionTemplate.executeWithoutResult(s -> {
                var toDelete = findExtIdsElderThan(size, theOffset);
                toDelete.forEach(userIdToDelete -> events.add(eventService.convertProfileDeleted(userIdToDelete)));
                userAccountRepository.deleteAllById(toDelete);
                getLogger().info("Deleted users with ids {} from database which were removed from {}", toDelete, getName());
                shouldContinue.set(toDelete.size() == size);
            });
            sendEvents();
        }
    }
    protected abstract UserAccount prepareUserAccountForInsert(T t);

    protected abstract Pair<UserAccount, Boolean> applyUpdateToUserAccount(T entity, UserAccount userAccount);

    protected abstract String getName();

    protected abstract List<UserAccount> findByExtId(Collection<String> extIds);

    protected abstract UserAccount setSyncTime(UserAccount userAccount);

    protected abstract void batchSetSyncTime(Set<String> toUpdateSetExtSyncTime);

    protected abstract Optional<UserAccount> findByExtUserId(List<UserAccount> dbChunk, String extUserId);

    protected abstract List<Long> findExtIdsElderThan(int limit, int theOffset);

    protected abstract ConflictResolveStrategy getConflictResolvingStrategy();

    protected abstract String getRenamingPrefix();
}
