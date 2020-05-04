package com.github.nkonev.blog.config;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.MediaType;

import java.util.List;

@Configuration
@ConfigurationProperties(prefix="custom.image")
public class ImageConfigImpl implements ImageConfig {

    private List<MediaType> allowedMimeTypes;

    private long maxBytes;

    private long maxAge;

    public List<MediaType> getAllowedMimeTypes() {
        return allowedMimeTypes;
    }

    public void setAllowedMimeTypes(List<MediaType> allowedMimeTypes) {
        this.allowedMimeTypes = allowedMimeTypes;
    }

    public long getMaxBytes() {
        return maxBytes;
    }

    public long getMaxAge() {
        return maxAge;
    }

    public void setMaxBytes(long maxBytes) {
        this.maxBytes = maxBytes;
    }

    public void setMaxAge(long maxAge) {
        this.maxAge = maxAge;
    }
}
