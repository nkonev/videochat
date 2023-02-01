package com.github.nkonev.aaa.repository.jdbc;

import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.PagingAndSortingRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

@Repository
public interface UserAccountRepository extends PagingAndSortingRepository<UserAccount, Long> {

    Optional<UserAccount> findByUsername(String username);

    Optional<UserAccount> findByEmail(String email);

    // https://jira.spring.io/projects/DATAJDBC/issues/DATAJDBC-101?filter=allopenissues
    @Query("select * from users u where u.username ilike :userName order by id limit :limit offset :offset")
    List<UserAccount> findByUsernameContainsIgnoreCase(@Param("limit")long limit, @Param("offset")long offset, @Param("userName")String login);

    @Query("select count(id) from users u where u.username ilike :userName")
    long findByUsernameContainsIgnoreCaseCount(@Param("userName")String searchString);

    @Query("select * from users u where u.username ilike :userName and id not in (:excludingUserIds) order by id limit :limit offset :offset")
    List<UserAccount> findByUsernameContainsIgnoreCaseAndIdNotIn(@Param("limit")int pageSize, @Param("offset")long offset, @Param("userName") String forDbSearch, @Param("excludingUserIds")List<Long> userIds);

    @Query("select count(id) from users u where u.username ilike :userName and id not in (:excludingUserIds)")
    long findByUsernameContainsIgnoreCaseAndIdNotInCount(@Param("userName")String searchString, @Param("excludingUserIds")List<Long> userIds);

    @Query("select * from users u where u.username ilike :userName and id in (:includingUserIds) order by id limit :limit offset :offset")
    List<UserAccount> findByUsernameContainsIgnoreCaseAndIdIn(@Param("limit")int pageSize, @Param("offset")long offset, @Param("userName") String forDbSearch, @Param("includingUserIds")List<Long> userIds);

    @Query("select count(id) from users u where u.username ilike :userName and id in (:includingUserIds)")
    long findByUsernameContainsIgnoreCaseAndIdInCount(@Param("userName")String searchString, @Param("includingUserIds")List<Long> userIds);

    Optional<UserAccount> findByOauth2IdentifiersFacebookId(String facebookId);

    Optional<UserAccount> findByOauth2IdentifiersVkontakteId(String vkontakteId);

    Optional<UserAccount> findByOauth2IdentifiersGoogleId(String googleId);

    Optional<UserAccount> findByOauth2IdentifiersKeycloakId(String keycloakId);

    @Modifying
    @Query("update users set last_login_date_time = :newLastLoginDateTime where username = :userName")
    void updateLastLogin(@Param("userName") String username, @Param("newLastLoginDateTime") LocalDateTime localDateTime);

    List<UserAccount> findByIdInOrderById(List<Long> userIds);
}
