package name.nkonev.aaa.services;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictBy;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.*;

import static name.nkonev.aaa.Constants.LDAP_CONFLICT_PREFIX;

@Service
public class ConflictService {

    @Autowired
    private CheckService checkService;

    @Autowired
    private AaaProperties aaaProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(ConflictService.class);

    public void process(UserAccount newUser, ConflictResolvingActions conflictResolvingActions) {
        process(List.of(newUser), conflictResolvingActions);
    }

    // we suppose that vast majority of users will not have any conflicts ...
    public void process(Collection<UserAccount> newUsers, ConflictResolvingActions conflictResolvingActions) {
        if (newUsers.isEmpty()) {
            return;
        }
        var conflictingByUsernamesOldUsers = checkService.checkLogins(newUsers.stream().map(UserAccount::username).toList());
        var conflictingEmailsOldUsers = checkService.checkEmails(newUsers.stream().map(UserAccount::email).toList());

        var nonConflictingUsers = new ArrayList<>(newUsers);
        nonConflictingUsers.removeIf(u -> conflictingByUsernamesOldUsers.containsKey(u.username()));
        nonConflictingUsers.removeIf(u -> conflictingEmailsOldUsers.containsKey(u.email()));

        // ... so we save them in batch
        if (!nonConflictingUsers.isEmpty()) {
            conflictResolvingActions.saveUsers(nonConflictingUsers);
        }

        if (nonConflictingUsers.size() == newUsers.size()) {
            LOGGER.debug("No conflicting users in this batch");
            return;
        }

        var conflictingNewUsers = new ArrayList<UserAccount>();
        for (var inputUser : newUsers) {
            // prevent duplication
            if (conflictingByUsernamesOldUsers.containsKey(inputUser.username())) {
                conflictingNewUsers.add(inputUser);
            } else if (conflictingEmailsOldUsers.containsKey(inputUser.email())) {
                conflictingNewUsers.add(inputUser);
            }
        }

        LOGGER.info("For new users {} found conflicting old users: by username {}, by email {}", conflictingNewUsers, conflictingByUsernamesOldUsers.values(), conflictingEmailsOldUsers.values());

        for (var newUser : conflictingNewUsers) {
            Map<ConflictBy, UserAccount> conflictBy = new HashMap<>(); // can be different old users

            var oldUserConflictingByUsername = conflictingByUsernamesOldUsers.get(newUser.username());
            if (oldUserConflictingByUsername != null) {
                conflictBy.put(ConflictBy.USERNAME, oldUserConflictingByUsername);
            }
            // can be a conflict by both username and email
            var oldUserConflictingByEmail = conflictingEmailsOldUsers.get(newUser.email());
            if (oldUserConflictingByEmail != null) {
                conflictBy.put(ConflictBy.EMAIL, oldUserConflictingByEmail);
            }

            solveConflict(conflictResolvingActions, newUser, conflictBy);
        }
    }


    private void solveConflict(ConflictResolvingActions conflictResolvingActions, UserAccount newUser, Map<ConflictBy, UserAccount> conflictBy) {
        switch (aaaProperties.ldap().resolveConflictsStrategy()) {
            case IGNORE:
                LOGGER.info("Skipping importing an user {} with conflicting by {}", newUser, conflictBy.keySet());
                return;
            case WRITE_NEW_AND_REMOVE_OLD:
                conflictBy.forEach((cb, oldUser) -> {
                    LOGGER.info("Removing old conflicting by {} user {}", cb, oldUser);
                    conflictResolvingActions.removeUser(oldUser);
                });
                LOGGER.info("Saving new user {}", newUser);
                conflictResolvingActions.saveUser(newUser);
                return;
            case WRITE_NEW_AND_RENAME_OLD:
                conflictBy.forEach((cb, oldUser) -> {
                    switch (cb) {
                        case USERNAME:
                            var rl = LDAP_CONFLICT_PREFIX + oldUser.username();
                            LOGGER.info("Saving old conflicting user {} with renamed login {}", oldUser, rl);
                            var oldUserU = oldUser.withUsername(rl);
                            conflictResolvingActions.saveUser(oldUserU);
                            break;
                        case EMAIL:
                            var re = LDAP_CONFLICT_PREFIX + oldUser.email();
                            LOGGER.info("Saving old conflicting user {} with renamed email {}", oldUser, re);
                            var oldUserE = oldUser.withEmail(re);
                            conflictResolvingActions.saveUser(oldUserE);
                            break;
                        default:
                            throw new IllegalStateException("Unexpected conflict resolving strategy: " + cb);
                    }
                });
                LOGGER.info("Saving new user {}", newUser);
                conflictResolvingActions.saveUser(newUser);
                return;
            default:
                throw new IllegalStateException("Missed action for conflict strategy: " + aaaProperties.ldap().resolveConflictsStrategy());
        }
    }

}
