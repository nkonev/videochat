package com.github.nkonev.blog.services.port;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.elasticsearch.rest.RestClientProperties;
import org.springframework.stereotype.Service;
import org.springframework.util.StringUtils;

import javax.annotation.PostConstruct;

@Service(ElasticsearchPortChecker.NAME)
public class ElasticsearchPortChecker extends AbstractPortChecker {

    public static final String NAME="elasticsearchPortChecker";

    @Value("${port.check.elasticsearch.max.count:512}")
    private int maxCount;

    @Autowired
    private RestClientProperties elasticsearchProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(ElasticsearchPortChecker.class);

    @PostConstruct
    public void checkPorts(){
        LOGGER.info("Will check elasticsearch connection");

        for (String uri: elasticsearchProperties.getUris()) {
            LOGGER.info("Checking elasticsearch http uri {}", uri);

            String[] segments = StringUtils.delimitedListToStringArray(uri, ":");
            String host = segments[0].trim();
            int port = Integer.valueOf(segments[1].trim());

            check(maxCount, host, port);
        }

        LOGGER.info("Elasticsearch connection is ok");
    }

    @Override
    protected Logger getLogger() {
        return LOGGER;
    }
}
