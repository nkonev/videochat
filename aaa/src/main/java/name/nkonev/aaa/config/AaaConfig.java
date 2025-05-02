package name.nkonev.aaa.config;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.utils.ResourceUtils;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;

@EnableConfigurationProperties(AaaProperties.class)
@Configuration
public class AaaConfig {

    @Value("classpath:/static/git.json")
    private Resource resource;

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaConfig.class);

    @PostConstruct
    public void printVersion() {
        if(resource.exists()){
            String text = ResourceUtils.stringFromResource(resource);
            LOGGER.info("Version {}", text);
        } else {
            LOGGER.info("Version not exists");
        }
    }

    @PreDestroy
    public void preDestroy() {
        LOGGER.info("Shutting down aaa");
    }
}
