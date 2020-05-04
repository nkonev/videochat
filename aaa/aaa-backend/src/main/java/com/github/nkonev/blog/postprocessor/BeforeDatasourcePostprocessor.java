package com.github.nkonev.blog.postprocessor;

import com.github.nkonev.blog.services.port.JdbcPortChecker;
import org.springframework.boot.autoconfigure.AbstractDependsOnBeanFactoryPostProcessor;
import org.springframework.context.annotation.Configuration;

import javax.sql.DataSource;

@Configuration
public class BeforeDatasourcePostprocessor extends AbstractDependsOnBeanFactoryPostProcessor {

	public BeforeDatasourcePostprocessor() {
        super(DataSource.class, JdbcPortChecker.NAME);
    }
}
