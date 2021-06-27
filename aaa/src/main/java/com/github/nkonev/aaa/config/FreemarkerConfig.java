package com.github.nkonev.aaa.config;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.view.freemarker.FreeMarkerConfigurer;
import javax.annotation.PostConstruct;
import java.util.Arrays;

/**
 * https://vorba.ch/2018/spring-boot-freemarker-security-jsp-taglib.html
 */
@Configuration
public class FreemarkerConfig {

    @Autowired
    private FreeMarkerConfigurer freeMarkerConfigurer;

    @PostConstruct
    public void postConstruct() {
        freeMarkerConfigurer.getTaglibFactory().setClasspathTlds(Arrays.asList("/META-INF/security.tld", "/META-INF/spring-form.tld"));
        freeMarkerConfigurer.setPreferFileSystemAccess(false);
    }

}
