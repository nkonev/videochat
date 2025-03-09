package name.nkonev.aaa.services;

import name.nkonev.aaa.config.properties.ConflictBy;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.*;

@Service
public class ConflictService {

    @Autowired
    private CheckService checkService;

    public enum PotentiallyConflictingAction {
        INSERT,
        UPDATE
    }

    private static final Logger LOGGER = LoggerFactory.getLogger(ConflictService.class);

    public void process(String renamingPrefix, ConflictResolveStrategy resolveConflictsStrategy, PotentiallyConflictingAction action, UserAccount newUser, ConflictResolvingActions conflictResolvingActions, List<EventWrapper<?>> eventsContainer) {
        process(renamingPrefix, resolveConflictsStrategy, action, List.of(newUser), conflictResolvingActions, eventsContainer);
    }

    // we suppose that vast majority of users will not have any conflicts ...
    public void process(String renamingPrefix, ConflictResolveStrategy resolveConflictsStrategy, PotentiallyConflictingAction action, Collection<UserAccount> newUsers, ConflictResolvingActions conflictResolvingActions, List<EventWrapper<?>> eventsContainer) {
        if (newUsers.isEmpty()) {
            return;
        }
        var conflictingByLoginsOldUsers = checkService.checkLogins(newUsers.stream().map(UserAccount::login).toList());
        var conflictingEmailsOldUsers = checkService.checkEmails(newUsers.stream().map(UserAccount::email).toList());

        if (action == PotentiallyConflictingAction.UPDATE) {
            for (var nu : newUsers) {
                var conflByLogin = conflictingByLoginsOldUsers.get(nu.login());
                if (conflByLogin != null && conflByLogin.id().equals(nu.id())) { // remove myself
                    conflictingByLoginsOldUsers.remove(nu.login());
                }

                var conflByEmail = conflictingEmailsOldUsers.get(nu.email());
                if (conflByEmail != null && conflByEmail.id().equals(nu.id())) {
                    conflictingEmailsOldUsers.remove(nu.email()); // remove myself
                }
            }
        }

        var nonConflictingUsers = new ArrayList<>(newUsers);
        nonConflictingUsers.removeIf(u -> conflictingByLoginsOldUsers.containsKey(u.login()));
        nonConflictingUsers.removeIf(u -> conflictingEmailsOldUsers.containsKey(u.email()));

        // ... so we save them in batch
        if (!nonConflictingUsers.isEmpty()) {
            switch (action) {
                case INSERT:
                    conflictResolvingActions.insertUsers(nonConflictingUsers, eventsContainer);
                    break;
                case UPDATE:
                    conflictResolvingActions.updateUsers(nonConflictingUsers, eventsContainer);
                    break;
                default:
                    throw new IllegalStateException("Unexpected value: " + action);
            }
        }

        if (nonConflictingUsers.size() == newUsers.size()) {
            LOGGER.debug("No conflicting users in this batch");
            return;
        }

        var conflictingNewUsers = new ArrayList<UserAccount>();
        for (var inputUser : newUsers) {
            // prevent duplication
            if (conflictingByLoginsOldUsers.containsKey(inputUser.login())) {
                conflictingNewUsers.add(inputUser);
            } else if (conflictingEmailsOldUsers.containsKey(inputUser.email())) {
                conflictingNewUsers.add(inputUser);
            }
        }

        LOGGER.info("For new users {} found conflicting old users: by login {}, by email {}", conflictingNewUsers, conflictingByLoginsOldUsers.values(), conflictingEmailsOldUsers.values());

        for (var newUser : conflictingNewUsers) {
            Map<ConflictBy, UserAccount> conflictBy = new HashMap<>(); // can be different old users

            var oldUserConflictingByLogin = conflictingByLoginsOldUsers.get(newUser.login());
            if (oldUserConflictingByLogin != null) {
                conflictBy.put(ConflictBy.LOGIN, oldUserConflictingByLogin);
            }
            // can be a conflict by both login and email
            var oldUserConflictingByEmail = conflictingEmailsOldUsers.get(newUser.email());
            if (oldUserConflictingByEmail != null) {
                conflictBy.put(ConflictBy.EMAIL, oldUserConflictingByEmail);
            }

            solveConflict(renamingPrefix, resolveConflictsStrategy, action, conflictResolvingActions, newUser, conflictBy, eventsContainer);
        }
    }


    private void solveConflict(String renamingPrefix, ConflictResolveStrategy resolveConflictsStrategy, PotentiallyConflictingAction action, ConflictResolvingActions conflictResolvingActions, UserAccount newUser, Map<ConflictBy, UserAccount> conflictBy, List<EventWrapper<?>> eventsContainer) {
        switch (resolveConflictsStrategy) {
            case IGNORE:
                LOGGER.info("Skipping importing an user {} with conflicting by {}", newUser, conflictBy.keySet());
                return;
            case WRITE_NEW_AND_REMOVE_OLD:
                conflictBy.forEach((cb, oldUser) -> {
                    LOGGER.info("Removing old conflicting by {} user {}", cb, oldUser);
                    conflictResolvingActions.removeUser(oldUser, eventsContainer);
                });
                LOGGER.info("Saving new user {}", newUser);
                switch (action) {
                    case INSERT:
                        conflictResolvingActions.insertUser(newUser, eventsContainer);
                        break;
                    case UPDATE:
                        conflictResolvingActions.updateUser(newUser, eventsContainer);
                        break;
                    default:
                        throw new IllegalStateException("Unexpected value: " + action);
                }
                return;
            case WRITE_NEW_AND_RENAME_OLD:
                conflictBy.forEach((cb, oldUser) -> {
                    switch (cb) {
                        case LOGIN:
                            var rl = renamingPrefix + oldUser.login();
                            LOGGER.info("Saving old conflicting user {} with renamed login {}", oldUser, rl);
                            var oldUserU = oldUser.withLogin(rl);
                            conflictResolvingActions.updateUser(oldUserU, eventsContainer);
                            break;
                        case EMAIL:
                            var re = renamingPrefix + oldUser.email();
                            LOGGER.info("Saving old conflicting user {} with renamed email {}", oldUser, re);
                            var oldUserE = oldUser.withEmail(re);
                            conflictResolvingActions.updateUser(oldUserE, eventsContainer);
                            break;
                        default:
                            throw new IllegalStateException("Unexpected conflict resolving strategy: " + cb);
                    }
                });
                LOGGER.info("Saving new user {}", newUser);
                switch (action) {
                    case INSERT:
                        conflictResolvingActions.insertUser(newUser, eventsContainer);
                        break;
                    case UPDATE:
                        conflictResolvingActions.updateUser(newUser, eventsContainer);
                        break;
                    default:
                        throw new IllegalStateException("Unexpected value: " + action);
                }
                return;
            default:
                throw new IllegalStateException("Missed action for conflict strategy: " + resolveConflictsStrategy);
        }
    }

}
