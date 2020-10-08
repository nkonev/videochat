package com.github.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonAutoDetect;
import com.fasterxml.jackson.annotation.JsonTypeInfo;

import java.io.Serializable;

@JsonTypeInfo(use = JsonTypeInfo.Id.CLASS, include = JsonTypeInfo.As.PROPERTY, property = "@class")
@JsonAutoDetect(fieldVisibility = JsonAutoDetect.Visibility.ANY, getterVisibility = JsonAutoDetect.Visibility.NONE, setterVisibility = JsonAutoDetect.Visibility.NONE, isGetterVisibility = JsonAutoDetect.Visibility.NONE)
public class OAuth2IdentifiersDTO implements Serializable {
    private String facebookId;
    private String vkontakteId;
    private String googleId;

    public OAuth2IdentifiersDTO() {
    }

    public OAuth2IdentifiersDTO(String facebookId, String vkontakteId, String googleId) {
        this.facebookId = facebookId;
        this.vkontakteId = vkontakteId;
        this.googleId = googleId;
    }

    public String getFacebookId() {
        return facebookId;
    }

    public void setFacebookId(String facebookId) {
        this.facebookId = facebookId;
    }

    public String getVkontakteId() {
        return vkontakteId;
    }

    public void setVkontakteId(String vkontakteId) {
        this.vkontakteId = vkontakteId;
    }

    public String getGoogleId() {
        return googleId;
    }

    public void setGoogleId(String googleId) {
        this.googleId = googleId;
    }
}
