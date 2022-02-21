package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.AbstractUtTestRunner;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.autoconfigure.actuate.metrics.AutoConfigureMetrics;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import java.net.URI;
import static org.assertj.core.api.BDDAssertions.then;

@AutoConfigureMetrics
public class PrometheusTest extends AbstractUtTestRunner {

    @Value("${local.management.port}")
    protected int mgmtPort;

    private static final Logger LOGGER = LoggerFactory.getLogger(PrometheusTest.class);

    @Test
    public void testPrometheus() throws Exception {

        ResponseEntity<String> entity = this.testRestTemplate.exchange(
                RequestEntity.get(new URI("http://127.0.0.1:"+mgmtPort+"/actuator/prometheus"))
                        .accept(MediaType.TEXT_PLAIN).build(), String.class
        );

        then(entity.getStatusCode()).isEqualTo(HttpStatus.OK);
        then(entity.getBody()).contains("tomcat_sessions_");

    }

}
