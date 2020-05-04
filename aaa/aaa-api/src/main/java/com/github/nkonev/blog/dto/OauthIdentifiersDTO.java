package com.github.nkonev.blog.dto;

import java.io.Serializable;

public class OauthIdentifiersDTO implements Serializable {
    private String facebookId;
    private String vkontakteId;

    public OauthIdentifiersDTO() {
    }

    public OauthIdentifiersDTO(String facebookId, String vkontakteId) {
        this.facebookId = facebookId;
        this.vkontakteId = vkontakteId;
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
}
