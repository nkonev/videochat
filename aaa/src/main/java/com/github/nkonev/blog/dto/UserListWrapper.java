package com.github.nkonev.blog.dto;

import java.util.Collection;

public class UserListWrapper extends Wrapper<UserAccountDTO> {
    private boolean canManageSessions;

    public UserListWrapper() { }

    public UserListWrapper(Collection<UserAccountDTO> data, long totalCount, boolean canManageSessions) {
        super(data, totalCount);
        this.canManageSessions = canManageSessions;
    }

    public boolean isCanManageSessions() {
        return canManageSessions;
    }

    public void setCanManageSessions(boolean canManageSessions) {
        this.canManageSessions = canManageSessions;
    }
}
