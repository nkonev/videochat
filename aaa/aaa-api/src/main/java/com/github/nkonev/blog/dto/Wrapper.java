package com.github.nkonev.blog.dto;

import java.util.Collection;

public class Wrapper<T> {

    /**
     * total count
     */
    private long totalCount;

    private Collection<T> data;

    public Wrapper() { }

    public Wrapper(Collection<T> data, long totalCount) {
        this.data = data;
        this.totalCount = totalCount;
    }

    public Collection<T> getData() {
        return data;
    }

    public void setData(Collection<T> data) {
        this.data = data;
    }

    public long getTotalCount() {
        return totalCount;
    }

    public void setTotalCount(long totalCount) {
        this.totalCount = totalCount;
    }
}
