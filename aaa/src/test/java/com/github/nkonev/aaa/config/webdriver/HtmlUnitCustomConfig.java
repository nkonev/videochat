package com.github.nkonev.aaa.config.webdriver;

import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Configuration;

@Configuration
@EnableConfigurationProperties(HtmlUnitProperties.class)
public class HtmlUnitCustomConfig {
}
