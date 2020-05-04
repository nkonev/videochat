package com.github.nkonev.blog.repository.jdbc;

import com.github.nkonev.blog.entity.jdbc.RuntimeSettings;
import org.springframework.data.repository.CrudRepository;
import org.springframework.stereotype.Repository;

@Repository
public interface RuntimeSettingsRepository extends CrudRepository<RuntimeSettings, String> {

}
