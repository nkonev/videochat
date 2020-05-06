package com.github.nkonev.blog.config;

import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.jdbc.core.convert.JdbcCustomConversions;
import org.springframework.data.jdbc.core.mapping.JdbcMappingContext;
import org.springframework.data.jdbc.repository.config.AbstractJdbcConfiguration;
import org.springframework.data.jdbc.repository.config.EnableJdbcRepositories;
import org.springframework.data.relational.core.mapping.NamingStrategy;
import org.springframework.data.relational.core.mapping.RelationalMappingContext;
import org.springframework.transaction.annotation.EnableTransactionManagement;

import java.util.Optional;

@Configuration
@EntityScan(basePackages = "com.github.nkonev.blog.entity.jdbc")
@EnableJdbcRepositories(basePackages = "com.github.nkonev.blog.repository.jdbc")
@EnableTransactionManagement
public class DbConfig extends AbstractJdbcConfiguration {

    @Bean
    @Override
    public JdbcMappingContext jdbcMappingContext(Optional<NamingStrategy> namingStrategy,
                                                 JdbcCustomConversions customConversions) {

        JdbcMappingContext mappingContext = super.jdbcMappingContext(namingStrategy, customConversions);
        mappingContext.setForceQuote(false);
        return mappingContext;
    }


}
