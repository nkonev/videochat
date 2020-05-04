package com.github.nkonev.blog.config;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.TimeUnit;

@Configuration
@ConfigurationProperties("custom.rendertron")
public class RendertronConfig {
    private String crawlerUserAgents;
    private String serviceUrl;
    private String ignoreExtensions;
    private List<String> blacklistPaths = Collections.singletonList("/error");
    private TimeUnit cacheExpireTimeUnit = TimeUnit.MINUTES;
    private long cacheExpire = 30;
    private String userAgent = "HeadlessChrome";

    public RendertronConfig() { }

    public String getCrawlerUserAgents() {
        return crawlerUserAgents;
    }

    public void setCrawlerUserAgents(String crawlerUserAgents) {
        this.crawlerUserAgents = crawlerUserAgents;
    }


    public String getServiceUrl() {
        return serviceUrl;
    }

    public void setServiceUrl(String serviceUrl) {
        this.serviceUrl = serviceUrl;
    }

    public String getIgnoreExtensions() {
        return ignoreExtensions;
    }

    public void setIgnoreExtensions(String ignoreExtensions) {
        this.ignoreExtensions = ignoreExtensions;
    }

    public TimeUnit getCacheExpireTimeUnit() {
        return cacheExpireTimeUnit;
    }

    public void setCacheExpireTimeUnit(TimeUnit cacheExpireTimeUnit) {
        this.cacheExpireTimeUnit = cacheExpireTimeUnit;
    }

    public long getCacheExpire() {
        return cacheExpire;
    }

    public void setCacheExpire(long cacheExpire) {
        this.cacheExpire = cacheExpire;
    }

    @Override
    public String toString() {
        return "PrerenderConfig{" +
                "crawlerUserAgents='" + crawlerUserAgents + '\'' +
                ", serviceUrl='" + serviceUrl + '\'' +
                ", ignoreExtensions='" + ignoreExtensions + '\'' +
                ", cacheExpireTimeUnit=" + cacheExpireTimeUnit +
                ", cacheExpire=" + cacheExpire +
                '}';
    }

    public List<String> getBlacklistPaths() {
        return blacklistPaths;
    }

    public void setBlacklistPaths(List<String> blacklistPaths) {
        this.blacklistPaths = blacklistPaths;
    }

    public String getUserAgent() {
        return userAgent;
    }

    public void setUserAgent(String userAgent) {
        this.userAgent = userAgent;
    }
}
