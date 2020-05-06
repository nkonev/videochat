package com.github.nkonev.blog.services;

import org.flywaydb.core.Flyway;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.flyway.FlywayMigrationStrategy;
import org.springframework.stereotype.Component;

@Component
public class FlywayDropFirstMigrationStrategy implements FlywayMigrationStrategy {

    @Value("${spring.flyway.drop-first:false}")
    private boolean dropFirst;

    private Logger LOGGER = LoggerFactory.getLogger(FlywayDropFirstMigrationStrategy.class);

    @Override
    public void migrate(Flyway flyway) {
        if (dropFirst){
            flyway.clean();
        }
        flyway.migrate();
    }
}
