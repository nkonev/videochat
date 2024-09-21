package name.nkonev.aaa.repository.jdbc;

import name.nkonev.aaa.dto.UserRole;
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

    List<UserAccount> findByUsernameInOrderById(List<String> logins);

    List<UserAccount> findByEmailInOrderById(List<String> emails);

    List<UserAccount> findByLdapIdInOrderById(Collection<String> strings);

    List<UserAccount> findByKeycloakIdInOrderById(Collection<String> strings);

    @Modifying
    @Query("update user_account set sync_ldap_date_time = :newSyncLdapDateTime where ldap_id in (:ldapUserIds)")
    void updateSyncLdapTime(Set<String> ldapUserIds, LocalDateTime newSyncLdapDateTime);

    @Modifying
    @Query("update user_account set sync_keycloak_date_time = :newSyncKeycloakDateTime where keycloak_id in (:keycloakUserIds)")
    void updateSyncKeycloakTime(Set<String> keycloakUserIds, LocalDateTime newSyncKeycloakDateTime);

    @Modifying
    @Query("update user_account set sync_keycloak_roles_date_time = :newSyncKeycloakRolesDateTime where keycloak_id in (:keycloakUserIds)")
    void updateSyncKeycloakRolesTime(Set<String> keycloakUserIds, LocalDateTime newSyncKeycloakRolesDateTime);

    @Query("select id from user_account where ldap_id is not null and sync_ldap_date_time < :currTime limit :limit offset :offset")
    List<Long> findByLdapIdElderThan(LocalDateTime currTime, int limit, int offset);

    @Query("select id from user_account where keycloak_id is not null and sync_keycloak_date_time < :currTime limit :limit offset :offset")
    List<Long> findByKeycloakIdElderThan(LocalDateTime currTime, int limit, int offset);

    @Query("select count (*) from user_account where ldap_id is not null")
    long countLdap();

    @Query("select count (*) from user_account where keycloak_id is not null")
    long countKeycloak();

    @Query("select * from user_account where keycloak_id is not null and sync_keycloak_roles_date_time < :currTime limit :limit offset :offset")
    List<UserAccount> findByKeycloakIdRolesElderThan(LocalDateTime currTime, int limit, int offset);
}
