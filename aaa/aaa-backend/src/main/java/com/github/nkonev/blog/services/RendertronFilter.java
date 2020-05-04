package com.github.nkonev.blog.services;

import com.github.nkonev.blog.config.CustomConfig;
import com.github.nkonev.blog.config.RendertronConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.core.io.Resource;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.GenericFilterBean;

import javax.annotation.PostConstruct;
import javax.servlet.FilterChain;
import javax.servlet.ServletException;
import javax.servlet.ServletRequest;
import javax.servlet.ServletResponse;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static com.github.nkonev.blog.utils.ResourceUtils.stringFromResourceOrNullIfNotExists;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyHtml;
import static com.github.nkonev.blog.utils.ServletUtils.getPath;
import static org.springframework.util.StringUtils.isEmpty;

@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
@ConditionalOnProperty(com.github.nkonev.blog.Constants.CUSTOM_RENDERTRON_ENABLE)
public class RendertronFilter extends GenericFilterBean {

    private static final String USER_AGENT = "User-Agent";
    @Autowired
    private RendertronConfig rendertronConfig;

    @Autowired
    private CustomConfig customConfig;

    @Autowired
    private SeoCacheService seoCacheService;

    @Value("${custom.seo.script:}")
    private Resource resource;

    private static final Logger LOGGER = LoggerFactory.getLogger(RendertronFilter.class);

    @PostConstruct
    public void print(){
        LOGGER.debug("Extensions to ignore by rendertron: {}", getExtensionsToIgnore());
        LOGGER.debug("Crawler userAgents: {}", getCrawlerUserAgents());
    }

    private List<String> getCrawlerUserAgents() {
        List<String> crawlerUserAgents = new ArrayList<String>(Arrays.asList("baiduspider",
                "facebookexternalhit", "twitterbot", "rogerbot", "linkedinbot", "embedly", "quora link preview",
                "showyoubo", "outbrain", "pinterest", "developers.google.com/+/web/snippet", "slackbot", "vkShare",
                "W3C_Validator", "redditbot", "Applebot", "yandex", "Googlebot"));
        final String crawlerUserAgentsFromConfig = rendertronConfig.getCrawlerUserAgents();
        if (!isEmpty(crawlerUserAgentsFromConfig)) {
            crawlerUserAgents.addAll(Arrays.asList(crawlerUserAgentsFromConfig.trim().split(",")));
        }

        return crawlerUserAgents;
    }

    private List<String> getExtensionsToIgnore() {
        List<String> extensionsToIgnore = new ArrayList<String>(Arrays.asList(".js", ".json", ".css", ".xml", ".less", ".png", ".jpg",
                ".jpeg", ".gif", ".pdf", ".doc", ".txt", ".ico", ".rss", ".zip", ".mp3", ".rar", ".exe", ".wmv",
                ".doc", ".avi", ".ppt", ".mpg", ".mpeg", ".tif", ".wav", ".mov", ".psd", ".ai", ".xls", ".mp4",
                ".m4a", ".swf", ".dat", ".dmg", ".iso", ".flv", ".m4v", ".torrent", ".woff", ".ttf"));
        final String extensionsToIgnoreFromConfig = rendertronConfig.getIgnoreExtensions();
        if (!isEmpty(extensionsToIgnoreFromConfig)) {
            extensionsToIgnore.addAll(Arrays.asList(extensionsToIgnoreFromConfig.trim().split(",")));
        }

        return extensionsToIgnore;
    }

    private boolean isInSearchUserAgent(final String userAgent) {
        if (userAgent == null){ return false;}
        for(String item: getCrawlerUserAgents()){
            if (userAgent.toLowerCase().contains(item.toLowerCase())){
                return true;
            }
        }
        return false;
    }

    private boolean isInResources(final String url) {
        for(String item: getExtensionsToIgnore()){
            if ((url.indexOf('?') >= 0 ? url.substring(0, url.indexOf('?')) : url)
                    .toLowerCase().endsWith(item)){
                return true;
            }
        }
        return false;
    }

    private boolean isInBlackList(String path) {
        if (rendertronConfig.getBlacklistPaths() == null) {
            return false;
        } else {
            return rendertronConfig.getBlacklistPaths().contains(path);
        }
    }

    @Override
    public void doFilter(ServletRequest servletRequest, ServletResponse servletResponse, FilterChain filterChain) throws IOException, ServletException {
        final HttpServletRequest request = (HttpServletRequest) servletRequest;
        final HttpServletResponse response = (HttpServletResponse) servletResponse;

        if (shouldUseRendertron(request)) {
            final String key = getRedisKeyHtml(request);
            String value = seoCacheService.getHtmlFromCache(key);

            if (value==null) {
                value = seoCacheService.rewriteCachedPage(request);
            }
            value = injectSeoScripts(value, request); // for Yandex verification
            final String userAgent = request.getHeader(USER_AGENT);
            LOGGER.info("Responding cached rendered page '{}' for User-Agent '{}'", getPath(request), userAgent);
            response.setHeader("Content-Type", "text/html; charset=utf-8");
            response.getWriter().print(value);
            return;
        }

        filterChain.doFilter(servletRequest, servletResponse);
    }

    public boolean shouldUseRendertron(HttpServletRequest request) {
        final String userAgent = request.getHeader(USER_AGENT);
        final String url = request.getRequestURL().toString();
        final String path = getPath(request);

        final boolean isInSearchUserAgent = isInSearchUserAgent(userAgent);
        final boolean notIsInResources = !isInResources(url);
        final boolean notIsInBlackList = !isInBlackList(path);
        final boolean shouldUseRendertron = isInSearchUserAgent && notIsInResources && notIsInBlackList;
        LOGGER.debug("shouldUseRendertron={} := (isInSearchUserAgent('{}')={} && !isInResources('{}')={} && !isInBlackList('{}')={})",
                shouldUseRendertron, userAgent, isInSearchUserAgent, url, notIsInResources, path, notIsInBlackList);
        return shouldUseRendertron;
    }

    /**
     * Intended for skip script rendering for prerender/rendertron
     * @param request
     * @return
     */
    public boolean shouldRenderSeoScript(HttpServletRequest request){
        final String userAgent = request.getHeader(USER_AGENT);
        return !isPrerenderUserAgent(userAgent);
    }

    private boolean isPrerenderUserAgent(String userAgent) {
        return userAgent != null && userAgent.contains(rendertronConfig.getUserAgent());
    }

    public String getSeoScript() {
        return stringFromResourceOrNullIfNotExists(resource);
    }

    private String injectSeoScripts(String value, HttpServletRequest request) {
        if (shouldRenderSeoScript(request)) {
            String maybeSeoScript = getSeoScript();
            if (maybeSeoScript != null) {
                value = value.replaceFirst("</head>", maybeSeoScript + "</head>");
            }
        }
        return value;
    }

}
