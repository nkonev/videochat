package name.nkonev.aaa.config;

import jakarta.annotation.PostConstruct;
import name.nkonev.aaa.config.properties.AaaProperties;
import org.apache.catalina.Valve;
import org.apache.catalina.filters.RequestDumperFilter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.autoconfigure.web.ServerProperties;
import org.springframework.boot.web.client.RestTemplateBuilder;
import org.springframework.boot.web.embedded.tomcat.TomcatServletWebServerFactory;
import org.springframework.boot.web.servlet.FilterRegistrationBean;
import org.springframework.boot.web.servlet.server.ServletWebServerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.MediaType;
import org.springframework.http.client.JdkClientHttpRequestFactory;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.servlet.config.annotation.ContentNegotiationConfigurer;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

import java.io.File;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    private static final Logger LOGGER = LoggerFactory.getLogger(WebConfig.class);

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private ServerProperties serverProperties;

    /**
     *  https://spring.io/blog/2013/05/11/content-negotiation-using-spring-mvc
     */
    @Override
    public void configureContentNegotiation(ContentNegotiationConfigurer configurer) {
        configurer.defaultContentType(MediaType.APPLICATION_JSON);
    }

    @PostConstruct
    public void log(){
        LOGGER.info("api url: {}, frontend url: {}", aaaProperties.apiUrl(), aaaProperties.frontendUrl());
    }

    @Bean
    public RestTemplate restTemplate() {
        return new RestTemplateBuilder()
                .connectTimeout(aaaProperties.httpClient().connectTimeout())
                .readTimeout(aaaProperties.httpClient().readTimeout())
                .requestFactory(JdkClientHttpRequestFactory.class)
                .build();
    }

    @ConditionalOnProperty("custom.request.dump")
    @Bean
    public FilterRegistrationBean<?> requestDumperFilter() {
        var registration = new FilterRegistrationBean<>();
        var requestDumperFilter = new RequestDumperFilter();
        registration.setFilter(requestDumperFilter);
        registration.addUrlPatterns("/*");
        return registration;
    }

    // see https://github.com/spring-projects/spring-boot/issues/14302#issuecomment-418712080 if you want to customize management tomcat
    @Bean
    public ServletWebServerFactory servletContainer(Valve... valves) {
        var tomcat = new TomcatServletWebServerFactory();
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
