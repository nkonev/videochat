package com.github.nkonev.blog.config;

import ch.qos.logback.access.servlet.TeeFilter;
import ch.qos.logback.access.tomcat.LogbackValve;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.web.servlet.FilterRegistrationBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@ConditionalOnProperty(value = "custom.access.log.enable", matchIfMissing = false)
@Configuration
public class AccessLogConfig {
    @Bean
    public LogbackValve valve(){
        LogbackValve logbackValve = new LogbackValve();
        logbackValve.setFilename("config/logback-access.xml");
        logbackValve.setAsyncSupported(true);
        return logbackValve;
    }

    @Bean
    public FilterRegistrationBean registration() {
        FilterRegistrationBean registration = new FilterRegistrationBean(new TeeFilter());
        registration.setOrder(-200); // set lesser value to be before SpringSecurityFilterChain for prevent output AccessDeniedException to stderr
        registration.setEnabled(true);
        return registration;
    }

}
