package com.github.nkonev.blog.dto;

public class LockDTO {
    private long userId;
    private boolean lock;

    public LockDTO() { }

    public LockDTO(long userId, boolean lock) {
        this.userId = userId;
        this.lock = lock;
    }

    public long getUserId() {
        return userId;
    }

    public void setUserId(long userId) {
        this.userId = userId;
    }

    public boolean isLock() {
        return lock;
    }

    public void setLock(boolean lock) {
        this.lock = lock;
    }
}
