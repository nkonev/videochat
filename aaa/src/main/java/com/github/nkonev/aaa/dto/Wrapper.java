package com.github.nkonev.aaa.dto;

import java.util.Collection;

public record Wrapper<T> (

    /**
     * total count
     */
    long totalCount,

    Collection<T> data
) { }