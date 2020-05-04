package com.github.nkonev.blog.services;

import com.github.nkonev.blog.Constants;
import com.github.nkonev.blog.config.CustomConfig;
import com.github.nkonev.blog.config.RendertronConfig;
import com.github.nkonev.blog.repository.jdbc.PostRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Primary;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import javax.servlet.http.HttpServletRequest;
import java.net.URI;
import java.net.URISyntaxException;

import static com.github.nkonev.blog.Constants.CUSTOM_RENDERTRON_ENABLE;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyForIndex;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyHtml;
import static com.github.nkonev.blog.utils.SeoCacheKeyUtils.getRedisKeyHtmlForPost;
import static com.github.nkonev.blog.utils.ServletUtils.getPath;
import static com.github.nkonev.blog.utils.ServletUtils.getQuery;

@ConditionalOnProperty(CUSTOM_RENDERTRON_ENABLE)
@Primary
@Service
public class SeoCacheServiceImpl implements SeoCacheService {

    @Autowired
    private RedisTemplate<String, String> redisTemplate;

    @Autowired
    private RendertronConfig rendertronConfig;

    @Autowired
    private CustomConfig customConfig;

    @Autowired
    private RestTemplate restTemplate;

    @Autowired
    private PostRepository postRepository;

    private static final Logger LOGGER = LoggerFactory.getLogger(SeoCacheServiceImpl.class);

    @Override
    public String getHtmlFromCache(String key) {
        return redisTemplate.opsForValue().get(key);
    }

    private void setHtml(String key, String value){
        redisTemplate.opsForValue().set(key, value);
        redisTemplate.expire(key, rendertronConfig.getCacheExpire(), rendertronConfig.getCacheExpireTimeUnit());
        LOGGER.info("Successfully set {} bytes html for key {}", value.getBytes()!=null?value.getBytes().length:0, key);
    }

    @Override
    public void removeAllPagesCache(Long postId) {
        if (postId != null){
            redisTemplate.delete(getRedisKeyHtmlForPost(postId));
            redisTemplate.delete(getRedisKeyForIndex());
        } else {
            redisTemplate.delete(getRedisKeyForIndex());
        }
    }


    /**
     *
     * @param path "/", "/post/3"
     * @param query "", "?a=b&c=d"
     * @return
     */
    private String getRendrered(String path, String query){
        final String rendertronUrl = rendertronConfig.getServiceUrl() + customConfig.getBaseUrl() + path + query;
        try {
            final RequestEntity<Void> requestEntity = RequestEntity.<Void>get(new URI(rendertronUrl))
                    .accept(MediaType.TEXT_HTML)
                    .build();
            LOGGER.info("Requesting {} from rendertron", rendertronUrl);
            final ResponseEntity<String> re = restTemplate.exchange(requestEntity, String.class);
            return re.getBody();

        } catch (URISyntaxException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public void refreshAllPagesCache(){
        LOGGER.info("Starting refreshing page cache");
        rewriteCachedIndex();

        postRepository.findPostIds().forEach(this::rewriteCachedPage);
        LOGGER.info("Finished refreshing page cache");
    }

    @Override
    public void rewriteCachedPage(Long postId) {
        if (postId == null) {return;}
        setHtml(getRedisKeyHtmlForPost(postId), getRendrered(Constants.Urls.POST + "/"+postId, ""));
    }

    @Override
    public void rewriteCachedIndex() {
        setHtml(getRedisKeyForIndex(), getRendrered("", ""));
    }

    @Override
    public String rewriteCachedPage(HttpServletRequest request) {
        final String key = getRedisKeyHtml(request);
        final String path = getPath(request);
        final String value = getRendrered(path, getQuery(request));
        setHtml(key, value);
        return value;
    }

}
