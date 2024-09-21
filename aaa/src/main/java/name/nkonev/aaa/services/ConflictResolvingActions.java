package name.nkonev.aaa.services;

import name.nkonev.aaa.entity.jdbc.UserAccount;

import java.util.Collection;

public interface ConflictResolvingActions {

    void insertUser(UserAccount userAccount);

    void updateUser(UserAccount userAccount);

    void removeUser(UserAccount userAccount);

    void insertUsers(Collection<UserAccount> users);
}
