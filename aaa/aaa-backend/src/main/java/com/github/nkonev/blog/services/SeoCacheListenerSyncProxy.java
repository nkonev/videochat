package com.github.nkonev.blog.services;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

@Component
public class SeoCacheListenerSyncProxy implements SeoCacheListenerProxy {

    @Autowired
    private SeoCacheService seoCacheService;

    public void rewriteCachedIndex() {
        seoCacheService.rewriteCachedIndex();
    }

    public void rewriteCachedPage(Long postId) {
        seoCacheService.rewriteCachedPage(postId);
    }
}
