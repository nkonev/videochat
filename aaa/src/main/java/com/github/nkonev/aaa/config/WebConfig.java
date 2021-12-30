package com.github.nkonev.aaa.config;

import org.apache.catalina.Context;
import org.apache.catalina.Valve;
import org.apache.tomcat.util.http.Rfc6265CookieProcessor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.web.ServerProperties;
import org.springframework.boot.web.client.RestTemplateBuilder;
import org.springframework.boot.web.embedded.tomcat.TomcatServletWebServerFactory;
import org.springframework.boot.web.servlet.server.ServletWebServerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.MediaType;
import org.springframework.util.StringUtils;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.servlet.config.annotation.ContentNegotiationConfigurer;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

import javax.annotation.PostConstruct;
import java.io.File;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    private static final Logger LOGGER = LoggerFactory.getLogger(WebConfig.class);

    @Autowired
    private CustomConfig customConfig;

    @Autowired
    private ServerProperties serverProperties;

    @Value("${cookie.same-site:}")
    private String sameSite;

    /**
     *  https://spring.io/blog/2013/05/11/content-negotiation-using-spring-mvc
     */
    @Override
    public void configureContentNegotiation(ContentNegotiationConfigurer configurer) {
        configurer.defaultContentType(MediaType.APPLICATION_JSON);
    }

    @PostConstruct
    public void log(){
        LOGGER.info("Base url: {}", customConfig.getBaseUrl());
    }

    @Bean
    public RestTemplate restTemplate() {
        return new RestTemplateBuilder()
                .setConnectTimeout(customConfig.getRestClientConnectTimeout())
                .setReadTimeout(customConfig.getRestClientReadTimeout())
                .build();
    }

    // see https://github.com/spring-projects/spring-boot/issues/14302#issuecomment-418712080 if you want to customize management tomcat
    @Bean
    public ServletWebServerFactory servletContainer(Valve... valves) {
        CustomizedTomcatServletWebServerFactory tomcat = new CustomizedTomcatServletWebServerFactory();
        tomcat.setSameSite(sameSite);
        if (valves != null) {
            tomcat.addContextValves(valves);
        }
        final File baseDir = serverProperties.getTomcat().getBasedir();
        if (baseDir!=null) {
            File docRoot = new File(baseDir, "document-root");
            docRoot.mkdirs();
            tomcat.setDocumentRoot(docRoot);
        }

        tomcat.setProtocol("org.apache.coyote.http11.Http11Nio2Protocol");

        return tomcat;
    }
}

class CustomizedTomcatServletWebServerFactory extends TomcatServletWebServerFactory {
    private String sameSite;

    @Override
    protected void postProcessContext(Context context) {
        if (StringUtils.hasLength(sameSite)) {
            Rfc6265CookieProcessor rfc6265Processor = new Rfc6265CookieProcessor();
            rfc6265Processor.setSameSiteCookies(sameSite);
            context.setCookieProcessor(rfc6265Processor);
        }
    }

    public void setSameSite(String sameSite) {
        this.sameSite = sameSite;
    }
}