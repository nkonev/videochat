package name.nkonev.aaa.config;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.databind.module.SimpleModule;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.fasterxml.jackson.datatype.jsr310.deser.LocalDateTimeDeserializer;
import com.fasterxml.jackson.datatype.jsr310.ser.LocalDateTimeSerializer;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import org.springframework.boot.jackson.autoconfigure.JsonMapperBuilderCustomizer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.time.format.DateTimeFormatterBuilder;
import java.time.temporal.ChronoField;

// TODO Migrate to Jackson 3
@Configuration
public class JacksonConfig {
    @Bean
    public JsonMapperBuilderCustomizer jc() {
        return
                builder -> {
                    // formatter configuration to make working parsing variable fraction milliseconds such as .1, .12, .123
                    // just using pattern "yyyy-MM-dd'T'HH:mm:ss.SSS'Z'" is not working since Jackson 2.19.0
                    DateTimeFormatter formatter = new DateTimeFormatterBuilder()
                            .appendPattern("yyyy-MM-dd'T'HH:mm:ss")
                            .parseLenient()
                            .appendFraction(ChronoField.MILLI_OF_SECOND, 3, 3, true)
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

                    builder.addModules(javaTimeModule, rejectUserAccountDetailsDTOModule);
                };
    }

}
