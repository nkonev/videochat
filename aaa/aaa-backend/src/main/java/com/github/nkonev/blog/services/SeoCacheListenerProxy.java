package com.github.nkonev.blog.services;

public interface SeoCacheListenerProxy {
    void rewriteCachedIndex();

    void rewriteCachedPage(Long postId);
}
