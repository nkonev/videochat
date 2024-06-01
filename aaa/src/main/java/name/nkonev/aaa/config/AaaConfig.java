package name.nkonev.aaa.config;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.databind.module.SimpleModule;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.utils.ResourceUtils;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;

@EnableConfigurationProperties(AaaProperties.class)
@Configuration
public class AaaConfig {

    @Autowired
    private ObjectMapper objectMapper;

    @Value("classpath:/static/git.json")
    private Resource resource;

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaConfig.class);

    @PostConstruct
    public void pc() {
        SimpleModule rejectUserAccountDetailsDTOModule = new SimpleModule("Reject serialize UserAccountDetailsDTO");
        rejectUserAccountDetailsDTOModule.addSerializer(UserAccountDetailsDTO.class, new JsonSerializer<UserAccountDetailsDTO>() {
            @Override
            public void serialize(UserAccountDetailsDTO value, JsonGenerator jgen, SerializerProvider provider){
                throw new RuntimeException("You shouldn't to serialize UserAccountDetailsDTO");
            }
        });
        objectMapper.registerModule(rejectUserAccountDetailsDTOModule);
    }

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
