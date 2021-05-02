package com.github.nkonev.aaa.entity.jdbc;

import com.github.nkonev.aaa.dto.UserRole;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Embedded;
import org.springframework.data.relational.core.mapping.Table;
import javax.validation.constraints.NotNull;
import java.time.LocalDateTime;

@Table("users")
public class UserAccount {
    @Id
    private Long id;
    private String username;
    private String password; // hash
    private String avatar; // avatar url
    private String avatarBig; // avatar url
    private boolean expired;
    private boolean locked;
    private boolean enabled; // synonym to "confirmed"
    private String email;

    @NotNull
    private CreationType creationType;

    @NotNull
    private UserRole role; // synonym to "authority"

    private LocalDateTime lastLoginDateTime;

    @Embedded(onEmpty = Embedded.OnEmpty.USE_EMPTY)
    private OAuth2Identifiers oauth2Identifiers = new OAuth2Identifiers();

    public UserAccount() { }

    public UserAccount(CreationType creationType, String username, String password, String avatar, String avatarBig,
                       boolean expired, boolean locked, boolean enabled,
                       UserRole role, String email, OAuth2Identifiers oauth2Identifiers) {
        this.creationType = creationType;
        this.username = username;
        this.password = password;
        this.avatar = avatar;
        this.avatarBig = avatarBig;
        this.expired = expired;
        this.locked = locked;
        this.enabled = enabled;
        this.role = role;
        this.email = email;
        if (oauth2Identifiers !=null){
            this.oauth2Identifiers = oauth2Identifiers;
        }
    }

    public OAuth2Identifiers getOauth2Identifiers() {
        if (oauth2Identifiers == null){
            oauth2Identifiers = new OAuth2Identifiers();
        }
        return oauth2Identifiers;
    }

    public void setOauth2Identifiers(OAuth2Identifiers oauth2Identifiers) {
        this.oauth2Identifiers = oauth2Identifiers;
    }

    public String getAvatarBig() {
        return avatarBig;
    }

    public void setAvatarBig(String avatarBig) {
        this.avatarBig = avatarBig;
    }

    public static class OAuth2Identifiers {
        private String facebookId;
        private String vkontakteId;
        private String googleId;

        public OAuth2Identifiers() {
        }

        public OAuth2Identifiers(String facebookId, String vkontakteId, String googleId) {
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

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getAvatar() {
        return avatar;
    }

    public void setAvatar(String avatar) {
        this.avatar = avatar;
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public boolean isExpired() {
        return expired;
    }

    public void setExpired(boolean expired) {
        this.expired = expired;
    }

    public boolean isLocked() {
        return locked;
    }

    public void setLocked(boolean locked) {
        this.locked = locked;
    }

    public boolean isEnabled() {
        return enabled;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public String getPassword() {
        return password;
    }

    public void setPassword(String password) {
        this.password = password;
    }

    public String getEmail() {
        return email;
    }

    public void setEmail(String email) {
        this.email = email;
    }

    public UserRole getRole() {
        return role;
    }

    public void setRole(UserRole role) {
        this.role = role;
    }

    public CreationType getCreationType() {
        return creationType;
    }

    public void setCreationType(CreationType creationType) {
        this.creationType = creationType;
    }

    public void setLastLoginDateTime(LocalDateTime lastLoginDateTime) {
        this.lastLoginDateTime = lastLoginDateTime;
    }

    public LocalDateTime getLastLoginDateTime() {
        return lastLoginDateTime;
    }
}
