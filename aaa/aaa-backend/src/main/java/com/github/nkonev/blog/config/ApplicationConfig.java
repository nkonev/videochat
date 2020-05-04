package com.github.nkonev.blog.config;

import com.github.nkonev.blog.dto.BaseApplicationDTO;
import org.springframework.boot.context.properties.ConfigurationProperties;

import java.util.ArrayList;
import java.util.List;

/**
 * Config for admin-defined applications.
 */
@ConfigurationProperties("custom")
public class ApplicationConfig {
    private boolean enableApplications = true;
    private List<BaseApplicationDTO> applications = new ArrayList<>();

    public List<BaseApplicationDTO> getApplications() {
        return applications;
    }

    public void setApplications(List<BaseApplicationDTO> applications) {
        this.applications = applications;
    }

    public boolean isEnableApplications() {
        return enableApplications;
    }

    public void setEnableApplications(boolean enableApplications) {
        this.enableApplications = enableApplications;
    }
}
