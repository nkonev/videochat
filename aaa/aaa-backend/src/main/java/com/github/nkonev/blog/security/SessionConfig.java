package com.github.nkonev.blog.security;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.session.web.http.CookieHttpSessionIdResolver;
import org.springframework.session.web.http.DefaultCookieSerializer;
import org.springframework.session.web.http.HttpSessionIdResolver;

@Configuration
public class SessionConfig {

  @Bean
  public HttpSessionIdResolver httpSessionIdResolver() {
    DefaultCookieSerializer cookieSerializer = new DefaultCookieSerializer();
    cookieSerializer.setUseBase64Encoding(false);
    CookieHttpSessionIdResolver cookieHttpSessionIdResolver = new CookieHttpSessionIdResolver();
    cookieHttpSessionIdResolver.setCookieSerializer(cookieSerializer);
    return cookieHttpSessionIdResolver;
  }

}
