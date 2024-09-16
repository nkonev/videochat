package name.nkonev.aaa.repository.jdbc;

import name.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.ListCrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.Collection;
import java.util.List;
import java.util.Optional;
import java.util.Set;

@Repository
public interface UserAccountRepository extends ListCrudRepository<UserAccount, Long> {

    Optional<UserAccount> findByUsername(String username);

    Optional<UserAccount> findByLdapId(String ldapId);

    Optional<UserAccount> findByEmail(String email);

    Optional<UserAccount> findByFacebookId(String facebookId);

    Optional<UserAccount> findByVkontakteId(String vkontakteId);

    Optional<UserAccount> findByGoogleId(String googleId);

    Optional<UserAccount> findByKeycloakId(String keycloakId);

    @Modifying
    @Query("update user_account set last_login_date_time = :newLastLoginDateTime where username = :userName")
    void updateLastLogin(@Param("userName") String username, @Param("newLastLoginDateTime") LocalDateTime localDateTime);

    List<UserAccount> findByIdInOrderById(List<Long> userIds);

    // here we intentionally set that deleted user exists
    @Query("select u.id from user_account u where u.id in (:userIds)")
    Set<Long> findUserIds(List<Long> userIds);

    List<UserAccount> findByLdapIdInOrderById(Collection<String> strings);
}
