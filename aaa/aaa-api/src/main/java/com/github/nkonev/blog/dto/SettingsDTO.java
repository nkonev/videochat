package com.github.nkonev.blog.dto;

import java.util.ArrayList;
import java.util.List;

public class SettingsDTO {
    private String header;

    private String subHeader;

    private String titleTemplate;

    private String imageBackground;

    private Boolean removeImageBackground;

    private String backgroundColor;

    private boolean canShowSettings;

    private boolean canShowApplications;

    private List<UserRole> availableRoles = new ArrayList<>();

    public SettingsDTO() { }

    public SettingsDTO(String header, String subHeader, String titleTemplate, String imageBackground, String backgroundColor, boolean canShowSettings, boolean enableApplications, List<UserRole> availableRoles) {
        this.header = header;
        this.subHeader = subHeader;
        this.titleTemplate = titleTemplate;
        this.imageBackground = imageBackground;
        this.backgroundColor = backgroundColor;
        this.canShowSettings = canShowSettings;
        this.canShowApplications = enableApplications;
        this.availableRoles = availableRoles;
    }

    public String getTitleTemplate() {
        return titleTemplate;
    }

    public void setTitleTemplate(String titleTemplate) {
        this.titleTemplate = titleTemplate;
    }

    public String getHeader() {
        return header;
    }

    public void setHeader(String header) {
        this.header = header;
    }

    public String getImageBackground() {
        return imageBackground;
    }

    public void setImageBackground(String imageBackground) {
        this.imageBackground = imageBackground;
    }

    public boolean isCanShowSettings() {
        return canShowSettings;
    }

    public void setCanShowSettings(boolean canShowSettings) {
        this.canShowSettings = canShowSettings;
    }

    public Boolean getRemoveImageBackground() {
        return removeImageBackground;
    }

    public void setRemoveImageBackground(Boolean removeImageBackground) {
        this.removeImageBackground = removeImageBackground;
    }

    public String getSubHeader() {
        return subHeader;
    }

    public void setSubHeader(String subHeader) {
        this.subHeader = subHeader;
    }

    public boolean isCanShowApplications() {
        return canShowApplications;
    }

    public void setCanShowApplications(boolean canShowApplications) {
        this.canShowApplications = canShowApplications;
    }

    public List<UserRole> getAvailableRoles() {
        return availableRoles;
    }

    public void setAvailableRoles(List<UserRole> availableRoles) {
        this.availableRoles = availableRoles;
    }

    public String getBackgroundColor() {
        return backgroundColor;
    }

    public void setBackgroundColor(String backgroundColor) {
        this.backgroundColor = backgroundColor;
    }
}
