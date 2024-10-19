package name.nkonev.aaa.services;

import name.nkonev.aaa.dto.EventWrapper;
import name.nkonev.aaa.entity.jdbc.UserAccount;

import java.util.Collection;
import java.util.List;

public interface ConflictResolvingActions {

    void insertUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer);

    void updateUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer);

    void removeUser(UserAccount userAccount, List<EventWrapper<?>> eventsContainer);

    void insertUsers(Collection<UserAccount> users, List<EventWrapper<?>> eventsContainer);

    void updateUsers(Collection<UserAccount> users, List<EventWrapper<?>> eventsContainer);
}
