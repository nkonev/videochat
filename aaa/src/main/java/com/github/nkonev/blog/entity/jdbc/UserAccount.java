package com.github.nkonev.blog.entity.jdbc;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.dto.UserRole;
import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Embedded;
import org.springframework.data.relational.core.mapping.Table;

import javax.validation.constraints.NotNull;
import java.time.LocalDateTime;

@Table(Constants.Schemas.AUTH+".users")
public class UserAccount {
    @Id
    private Long id;
    private String username;
    private String password; // hash
    private String avatar;
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
    private OauthIdentifiers oauthIdentifiers = new OauthIdentifiers();

    public UserAccount() { }

    public UserAccount(CreationType creationType, String username, String password, String avatar,
                       boolean expired, boolean locked, boolean enabled,
                       UserRole role, String email, OauthIdentifiers oauthIdentifiers) {
        this.creationType = creationType;
        this.username = username;
        this.password = password;
        this.avatar = avatar;
        this.expired = expired;
        this.locked = locked;
        this.enabled = enabled;
        this.role = role;
        this.email = email;
        if (oauthIdentifiers!=null){
            this.oauthIdentifiers = oauthIdentifiers;
        }
    }

    public OauthIdentifiers getOauthIdentifiers() {
        if (oauthIdentifiers == null){
            oauthIdentifiers = new OauthIdentifiers();
        }
        return oauthIdentifiers;
    }

    public void setOauthIdentifiers(OauthIdentifiers oauthIdentifiers) {
        this.oauthIdentifiers = oauthIdentifiers;
    }

    public static class OauthIdentifiers {
        private String facebookId;
        private String vkontakteId;

        public OauthIdentifiers() {
        }

        public OauthIdentifiers(String facebookId, String vkontakteId) {
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
