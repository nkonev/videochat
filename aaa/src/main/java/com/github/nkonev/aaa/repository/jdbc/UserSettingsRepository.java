package com.github.nkonev.aaa.repository.jdbc;

import com.github.nkonev.aaa.entity.jdbc.UserSettings;
import org.springframework.data.repository.PagingAndSortingRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface UserSettingsRepository extends PagingAndSortingRepository<UserSettings, Long> {
}
