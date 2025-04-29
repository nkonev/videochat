package name.nkonev.aaa.config;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.databind.module.SimpleModule;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.fasterxml.jackson.datatype.jsr310.deser.LocalDateTimeDeserializer;
import com.fasterxml.jackson.datatype.jsr310.ser.LocalDateTimeSerializer;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.utils.ResourceUtils;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.jackson.Jackson2ObjectMapperBuilderCustomizer;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.time.format.DateTimeFormatterBuilder;
import java.time.temporal.ChronoField;

@EnableConfigurationProperties(AaaProperties.class)
@Configuration
public class AaaConfig {

    @Value("classpath:/static/git.json")
    private Resource resource;

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaConfig.class);

    @Bean
    public Jackson2ObjectMapperBuilderCustomizer jc() {
        return
                builder -> {
                    // formatter configuration to make working parsing variable fraction milliseconds such as .1, .12, .123
                    // just using pattern "yyyy-MM-dd'T'HH:mm:ss.SSS'Z'" is not working since Jackson 2.19.0
                    DateTimeFormatter formatter = new DateTimeFormatterBuilder()
                            .appendPattern("yyyy-MM-dd'T'HH:mm:ss")
                            .parseLenient()
                            .appendLiteral('.')
                            .appendFraction(ChronoField.MILLI_OF_SECOND, 0, 3, false)
                            .appendLiteral('Z')
                            .toFormatter();
                    LocalDateTimeDeserializer dateTimeDeserializer = new LocalDateTimeDeserializer(formatter);
                    LocalDateTimeSerializer dateTimeSerializer = new LocalDateTimeSerializer(formatter);
                    JavaTimeModule javaTimeModule = new JavaTimeModule();
                    javaTimeModule.addDeserializer(LocalDateTime.class, dateTimeDeserializer);
                    javaTimeModule.addSerializer(LocalDateTime.class, dateTimeSerializer);

                    SimpleModule rejectUserAccountDetailsDTOModule = new SimpleModule("Reject serialize UserAccountDetailsDTO");
                    rejectUserAccountDetailsDTOModule.addSerializer(UserAccountDetailsDTO.class, new JsonSerializer<>() {
                        @Override
                        public void serialize(UserAccountDetailsDTO value, JsonGenerator jgen, SerializerProvider provider) {
                            throw new RuntimeException("You shouldn't to serialize UserAccountDetailsDTO");
                        }
                    });

                    builder.modules(javaTimeModule, rejectUserAccountDetailsDTOModule);
                };
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
