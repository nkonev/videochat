package com.github.nkonev.aaa.config;

import ch.qos.logback.access.servlet.TeeFilter;
import ch.qos.logback.access.tomcat.LogbackValve;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.web.servlet.FilterRegistrationBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@ConditionalOnProperty(value = "custom.access.log.enable", matchIfMissing = true)
@Configuration
public class AccessLogConfig {
    @Bean
    public LogbackValve valve(){
        LogbackValve logbackValve = new LogbackValve();
        logbackValve.setFilename("config/logback-access.xml");
        return logbackValve;
    }

    @Bean
    public FilterRegistrationBean registration() {
        FilterRegistrationBean registration = new FilterRegistrationBean(new TeeFilter());
        registration.setOrder(-2147483648); // set lesser value to be before SpringSecurityFilterChain for prevent output AccessDeniedException to stderr
        registration.setEnabled(true);
        return registration;
    }

}
