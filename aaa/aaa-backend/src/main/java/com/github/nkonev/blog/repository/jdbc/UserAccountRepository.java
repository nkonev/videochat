package com.github.nkonev.blog.repository.jdbc;

import com.github.nkonev.blog.entity.jdbc.UserAccount;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.CrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

@Repository
public interface UserAccountRepository extends CrudRepository<UserAccount, Long> {

    @Query("select * from auth.users u where u.username = :userName")
    Optional<UserAccount> findByUsername(@Param("userName") String username);

    @Query("select * from auth.users u where u.email = :email")
    Optional<UserAccount> findByEmail(@Param("email")String email);

    // https://jira.spring.io/projects/DATAJDBC/issues/DATAJDBC-101?filter=allopenissues
    @Query("select * from auth.users u where u.username ilike :userName order by id limit :limit offset :offset")
    List<UserAccount> findByUsernameContainsIgnoreCase(@Param("limit")long limit, @Param("offset")long offset, @Param("userName")String login);

    @Query("select count(id) from auth.users u where u.username ilike :userName")
    long findByUsernameContainsIgnoreCaseCount(@Param("limit")long limit, @Param("offset")long offset, @Param("userName")String searchString);

    @Query("select * from auth.users u where u.facebook_id = :i")
    Optional<UserAccount> findByOauthIdentifiersFacebookId(@Param("i")String facebookId);

    @Query("select * from auth.users u where u.vkontakte_id = :i")
    Optional<UserAccount> findByOauthIdentifiersVkontakteId(@Param("i")String vkontakteId);

    @Modifying
    @Query("update auth.users set last_login_date_time = :newLastLoginDateTime where username = :userName")
    void updateLastLogin(@Param("userName") String username, @Param("newLastLoginDateTime") LocalDateTime localDateTime);

}
