package com.github.nkonev.aaa.config;

import com.github.nkonev.aaa.controllers.ImageUserAvatarUploadController;

import java.time.Duration;
import java.util.concurrent.Executor;
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
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.scheduling.concurrent.ThreadPoolTaskExecutor;


@ConditionalOnProperty(value = "custom.tasks.enable", matchIfMissing = true)
@Configuration
public class TaskConfig {

    @Autowired
    private ImageUserAvatarUploadController imageUserAvatarUploadController;

    private static final String IMAGES_CLEAN_TASK = "imagesCleanTask";


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
        final int deletedAvatarImages = imageUserAvatarUploadController.clearAvatarImages();

        LOGGER_IMAGE_CLEAN_TASK.info("Cleared {} user avatar images;", deletedAvatarImages);
    }


}
