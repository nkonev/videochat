package com.github.nkonev.blog.services;

import com.github.nkonev.blog.AbstractUtTestRunner;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import java.net.URI;
import static org.assertj.core.api.BDDAssertions.then;

public class PrometheusTest extends AbstractUtTestRunner {
    private static final Logger LOGGER = LoggerFactory.getLogger(PrometheusTest.class);

    @Test
    public void testPrometheus() throws Exception {

        ResponseEntity<String> entity = this.testRestTemplate.exchange(
                RequestEntity.get(new URI("http://127.0.0.1:"+mgmtPort+"/actuator/prometheus"))
                        .accept(MediaType.TEXT_PLAIN).build(), String.class
        );

        then(entity.getStatusCode()).isEqualTo(HttpStatus.OK);
        then(entity.getBody()).contains("jvm_");

    }

}
