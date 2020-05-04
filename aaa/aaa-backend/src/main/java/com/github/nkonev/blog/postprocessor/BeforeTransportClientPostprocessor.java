package com.github.nkonev.blog.postprocessor;

import com.github.nkonev.blog.services.port.ElasticsearchPortChecker;
import org.elasticsearch.client.transport.TransportClient;
import org.springframework.boot.autoconfigure.AbstractDependsOnBeanFactoryPostProcessor;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.elasticsearch.client.TransportClientFactoryBean;

@Configuration
public class BeforeTransportClientPostprocessor extends AbstractDependsOnBeanFactoryPostProcessor {

	public BeforeTransportClientPostprocessor() {
        super(TransportClient.class, TransportClientFactoryBean.class, ElasticsearchPortChecker.NAME);
    }
}
