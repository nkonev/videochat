package com.github.nkonev.aaa.repository.jdbc;

import com.github.nkonev.aaa.dto.Language;
import com.github.nkonev.aaa.entity.jdbc.UserSettings;
import org.springframework.data.jdbc.repository.query.Modifying;
import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.ListCrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

@Repository
public interface UserSettingsRepository extends ListCrudRepository<UserSettings, Long> {

    @Modifying
    @Query("insert into user_settings(id) values(:userId)")
    void insertDefault(@Param("userId") long id);

    @Modifying
    @Query("update user_settings set smileys = :smileys where id = :userId")
    void updateSmileys(long userId, String[] smileys);

    @Modifying
    @Query("update user_settings set language = :language where id = :userId")
    void updateLanguage(long userId, Language language);

}
