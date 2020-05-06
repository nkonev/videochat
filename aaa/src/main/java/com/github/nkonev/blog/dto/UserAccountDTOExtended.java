package com.github.nkonev.blog.dto;

import java.time.LocalDateTime;

public class UserAccountDTOExtended extends UserAccountDTO {
    private static final long serialVersionUID = 6613889927056965939L;

    private DataDTO managementData;

    private boolean canLock;

    private boolean canDelete;

    private boolean canChangeRole;

    public UserAccountDTOExtended() { }

    public UserAccountDTOExtended(Long id, String login, String avatar, DataDTO managementData, LocalDateTime lastLoginDateTime, OauthIdentifiersDTO oauthIdentifiers, boolean canLock, boolean canDelete, boolean canChangeRole) {
        super(id, login, avatar, lastLoginDateTime, oauthIdentifiers);
        this.managementData = managementData;
        this.canDelete = canDelete;
        this.canLock = canLock;
        this.canChangeRole = canChangeRole;
    }

    public DataDTO getManagementData() {
        return managementData;
    }

    public void setManagementData(DataDTO managementData) {
        this.managementData = managementData;
    }

    public boolean isCanLock() {
        return canLock;
    }

    public void setCanLock(boolean canLock) {
        this.canLock = canLock;
    }

    public boolean isCanDelete() {
        return canDelete;
    }

    public void setCanDelete(boolean canDelete) {
        this.canDelete = canDelete;
    }

    public boolean isCanChangeRole() {
        return canChangeRole;
    }

    public void setCanChangeRole(boolean canChangeRole) {
        this.canChangeRole = canChangeRole;
    }


    public static class DataDTO {
        private boolean enabled;
        private boolean expired;
        private boolean locked;
        private UserRole role;

        public DataDTO(boolean enabled, boolean expired, boolean locked, UserRole role) {
            this.enabled = enabled;
            this.expired = expired;
            this.locked = locked;
            this.role = role;
        }

        public DataDTO() { }

        public boolean isEnabled() {
            return enabled;
        }

        public void setEnabled(boolean enabled) {
            this.enabled = enabled;
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

        public UserRole getRole() {
            return role;
        }

        public void setRole(UserRole role) {
            this.role = role;
        }
    }
}
