package com.github.nkonev.blog.utils;

import javax.servlet.http.HttpServletRequest;

import static com.github.nkonev.blog.Constants.Urls.POST;
import static com.github.nkonev.blog.utils.ServletUtils.getPath;
import static com.github.nkonev.blog.utils.ServletUtils.nullToEmpty;

public class SeoCacheKeyUtils {
    public static final String RENDERTRON_HTML = "rendertron:html:";

    public static String getRedisKeyHtml(HttpServletRequest clientRequest) {
        return RENDERTRON_HTML + getPath(clientRequest) + nullToEmpty(clientRequest.getQueryString());
    }

    public static String getRedisKeyHtmlForPost(Long postId) {
        return RENDERTRON_HTML + POST + "/" + postId;
    }

    public static String getRedisKeyForIndex(){
        return RENDERTRON_HTML;
    }

}
