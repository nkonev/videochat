package name.nkonev.aaa.services;

import name.nkonev.aaa.entity.jdbc.UserAccount;

import java.util.Collection;

public interface ConflictResolvingActions {

    void saveUser(UserAccount userAccount);

    void removeUser(UserAccount userAccount);

    void saveUsers(Collection<UserAccount> users);
}
