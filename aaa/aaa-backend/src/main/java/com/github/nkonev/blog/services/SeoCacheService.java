package com.github.nkonev.blog.services;


import javax.servlet.http.HttpServletRequest;

public interface SeoCacheService {
    String getHtmlFromCache(String key);

    void removeAllPagesCache(Long postId);

    void refreshAllPagesCache();

    void rewriteCachedPage(Long postId);

    void rewriteCachedIndex();

    String rewriteCachedPage(HttpServletRequest request);
}
