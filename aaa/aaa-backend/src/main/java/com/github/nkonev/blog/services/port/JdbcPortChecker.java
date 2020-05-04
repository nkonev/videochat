package com.github.nkonev.blog.services.port;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.stereotype.Service;

import javax.annotation.PostConstruct;
import java.net.URI;

@Service(JdbcPortChecker.NAME)
public class JdbcPortChecker extends AbstractPortChecker{

    public static final String NAME="jdbcPortChecker";

    @Value("${port.check.jdbc.max.count:64}")
    private int maxCount;

    @Autowired
    private DataSourceProperties dataSourceProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(JdbcPortChecker.class);

    @PostConstruct
    public void checkPorts(){
        LOGGER.info("Will check JDBC connection");
        String cleanURI = dataSourceProperties.getUrl().substring(5);
        URI uri = URI.create(cleanURI);
        check(maxCount, uri.getHost(), uri.getPort());
        LOGGER.info("JDBC connection is ok");
    }

    @Override
    protected Logger getLogger() {
        return LOGGER;
    }
}
