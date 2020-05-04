package com.github.nkonev.blog.config;

import com.github.nkonev.blog.controllers.ImagePostContentUploadController;
import com.github.nkonev.blog.controllers.ImagePostTitleUploadController;
import com.github.nkonev.blog.controllers.ImageSettingsUploadController;
import com.github.nkonev.blog.controllers.ImageUserAvatarUploadController;
import com.github.nkonev.blog.services.PostService;
import com.github.nkonev.blog.services.SeoCacheService;
import net.javacrumbs.shedlock.core.LockProvider;
import net.javacrumbs.shedlock.core.SchedulerLock;
import net.javacrumbs.shedlock.provider.redis.spring.RedisLockProvider;
import net.javacrumbs.shedlock.spring.ScheduledLockConfiguration;
import net.javacrumbs.shedlock.spring.ScheduledLockConfigurationBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.scheduling.concurrent.ThreadPoolTaskExecutor;
import java.time.Duration;
import java.util.concurrent.Executor;


@ConditionalOnProperty(value = "custom.tasks.enable", matchIfMissing = true)
@Configuration
public class TaskConfig {

    @Autowired
    private ImagePostContentUploadController imagePostContentUploadController;

    @Autowired
    private ImagePostTitleUploadController imagePostTitleUploadController;

    @Autowired
    private ImageUserAvatarUploadController imageUserAvatarUploadController;

    @Autowired
    private ImageSettingsUploadController imageSettingsUploadController;

    @Autowired
    private SeoCacheService seoCacheService;

    @Autowired
    private PostService postService;

    private static final String IMAGES_CLEAN_TASK = "imagesCleanTask";
    private static final String REFRESH_CACHE_CLEAN_TASK = "refreshRendertronCacheTask";
    private static final String REFRESH_INDEX_TASK = "refreshElasticsearchTask";

    public static final Logger LOGGER_IMAGE_CLEAN_TASK = LoggerFactory.getLogger(IMAGES_CLEAN_TASK);


    @Bean
    public LockProvider lockProvider(RedisConnectionFactory redisConnectionFactory) {
        return new RedisLockProvider(redisConnectionFactory);
    }

    @Bean
    public ScheduledLockConfiguration taskScheduler(
            LockProvider lockProvider,
            @Value("${custom.tasks.poolSize}") int poolSize,
            @Value("${custom.tasks.defaultLockAtMostForSec}") int defaultLockAtMostForSec,
            @Value("${custom.tasks.defaultLockAtLeastForSec}") int defaultLockAtLeastForSec
            ) {
        return ScheduledLockConfigurationBuilder
                .withLockProvider(lockProvider)
                .withPoolSize(poolSize)
                .withDefaultLockAtMostFor(Duration.ofSeconds(defaultLockAtMostForSec))
                .withDefaultLockAtLeastFor(Duration.ofSeconds(defaultLockAtLeastForSec))
                .build();
    }

    @Bean(name = "taskExecutor")
    public Executor asyncExecutor(
            @Value("${custom.async.corePoolSize:8}") int corePoolSize,
            @Value("${custom.async.maxPoolSize:8}") int maxPoolSize,
            @Value("${custom.async.queueCapacity:512}") int queueCapacity,
            @Value("${custom.async.threadNamePrefix:BlogAsync-}") String threadNamePrefix
    ) {
        ThreadPoolTaskExecutor executor = new ThreadPoolTaskExecutor();
        executor.setCorePoolSize(corePoolSize);
        executor.setMaxPoolSize(maxPoolSize);
        executor.setQueueCapacity(queueCapacity);
        executor.setThreadNamePrefix(threadNamePrefix);
        executor.initialize();
        return executor;
    }

    @Scheduled(cron = "${custom.tasks.images.clean.cron}")
    @SchedulerLock(name = IMAGES_CLEAN_TASK)
    public void cleanImages(){
        final int deletedPostContent = imagePostContentUploadController.clearPostContentImages();
        final int deletedPostTitles  = imagePostTitleUploadController.clearPostTitleImages();
        final int deletedAvatarImages = imageUserAvatarUploadController.clearAvatarImages();
        final int deletedSettingsImages = imageSettingsUploadController.clearSettingsImages();

        LOGGER_IMAGE_CLEAN_TASK.info("Cleared {} post content images(created before 1 day ago); {} post title images; {} user avatar images; {} settings images",
                deletedPostContent, deletedPostTitles, deletedAvatarImages, deletedSettingsImages);
    }


    @Scheduled(cron = "${custom.tasks.rendertron.cache.refresh.cron}")
    @SchedulerLock(name = REFRESH_CACHE_CLEAN_TASK)
    public void refreshCache(){
        seoCacheService.refreshAllPagesCache();
    }

    @Scheduled(cron = "${custom.tasks.elasticsearch.refresh.cron}")
    @SchedulerLock(name = REFRESH_INDEX_TASK)
    public void refreshIndex() {
        postService.refreshFulltextIndex(false);
    }
}
