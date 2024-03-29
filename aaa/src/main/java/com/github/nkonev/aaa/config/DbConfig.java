package com.github.nkonev.aaa.config;

import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.jdbc.repository.config.AbstractJdbcConfiguration;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.transaction.annotation.EnableTransactionManagement;

@Configuration
@EntityScan(basePackages = "com.github.nkonev.aaa.entity.jdbc")
@EnableJdbcRepositories(basePackages = "com.github.nkonev.aaa.repository.jdbc")
@EnableTransactionManagement
public class DbConfig extends AbstractJdbcConfiguration {

}
