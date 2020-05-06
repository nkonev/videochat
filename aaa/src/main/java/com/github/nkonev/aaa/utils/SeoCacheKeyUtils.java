package com.github.nkonev.aaa.utils;

import com.github.nkonev.aaa.Constants;

import javax.servlet.http.HttpServletRequest;

public class SeoCacheKeyUtils {
    public static final String RENDERTRON_HTML = "rendertron:html:";

    public static String getRedisKeyHtml(HttpServletRequest clientRequest) {
        return RENDERTRON_HTML + ServletUtils.getPath(clientRequest) + ServletUtils.nullToEmpty(clientRequest.getQueryString());
    }

    public static String getRedisKeyHtmlForPost(Long postId) {
        return RENDERTRON_HTML + Constants.Urls.POST + "/" + postId;
    }

    public static String getRedisKeyForIndex(){
        return RENDERTRON_HTML;
    }

}
