package com.github.nkonev.blog.services;

import org.springframework.stereotype.Service;

import javax.servlet.http.HttpServletRequest;

@Service
public class SeoCacheServiceNoop implements SeoCacheService {
    @Override
    public String getHtmlFromCache(String key) {
        return null;
    }

    @Override
    public void removeAllPagesCache(Long postId) {

    }

    @Override
    public void refreshAllPagesCache() {

    }

    @Override
    public void rewriteCachedPage(Long id) {

    }

    @Override
    public void rewriteCachedIndex() {

    }

    @Override
    public String rewriteCachedPage(HttpServletRequest request) {
        return null;
    }
}
