package name.nkonev.aaa.config;

import name.nkonev.aaa.config.properties.AaaProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.concurrent.ThreadPoolTaskScheduler;

@EnableScheduling
@Configuration
public class TaskConfiguration {

    @Bean(destroyMethod = "destroy")
    public ThreadPoolTaskScheduler createTaskScheduler(AaaProperties aaaProperties) {
        ThreadPoolTaskScheduler taskScheduler = new ThreadPoolTaskScheduler();
        taskScheduler.setThreadNamePrefix("aaa-tasks-");
        taskScheduler.setPoolSize(aaaProperties.schedulers().poolSize());
        // await in order not to terminate redis during task execution
        taskScheduler.setAwaitTerminationSeconds((int)aaaProperties.schedulers().awaitForTermination().toSeconds());
        return taskScheduler;
    }

}
