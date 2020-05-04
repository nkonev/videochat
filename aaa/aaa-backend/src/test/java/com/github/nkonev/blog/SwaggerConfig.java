package com.github.nkonev.blog;

import com.github.nkonev.blog.dto.SettingsDTO;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.util.AntPathMatcher;
import springfox.bean.validators.configuration.BeanValidatorPluginsConfiguration;
import springfox.documentation.builders.ApiInfoBuilder;
import springfox.documentation.service.ApiInfo;
import springfox.documentation.spi.DocumentationType;
import springfox.documentation.spring.web.plugins.Docket;
import springfox.documentation.swagger2.annotations.EnableSwagger2;
import java.io.InputStream;
import java.net.URL;
import java.net.URLStreamHandler;
import java.time.Duration;


// Created by nik on 28.05.17.

@EnableSwagger2
@Configuration
@Import(BeanValidatorPluginsConfiguration.class)
public class SwaggerConfig {

    @Bean
    public Docket restApi() {
        return new Docket(DocumentationType.SWAGGER_2)
                .ignoredParameterTypes(UserAccountDetailsDTO.class, URLStreamHandler.class, URL.class, Duration.class, InputStream.class, SettingsDTO.class)
                .apiInfo(apiInfo())
                .select()
                .paths(input ->
                        new AntPathMatcher().match("/api/**", input)
                )
                .build()
                .useDefaultResponseMessages(false);
    }

    private ApiInfo apiInfo() {
        return new ApiInfoBuilder()
                .title("Blog API Reference")
//                .description("Description")
//                .contact(new Contact("TestName", "http:/test-url.com", "test@test.de"))
//                .license("Apache 2.0")
//                .licenseUrl("http://www.apache.org/licenses/LICENSE-2.0.html")
//                .version("1.0.0")
                .build();
    }
}
