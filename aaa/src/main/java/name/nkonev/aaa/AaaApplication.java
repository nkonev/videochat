package name.nkonev.aaa;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"name.nkonev.aaa"}
)
public class AaaApplication {

    private static final Logger LOGGER = LoggerFactory.getLogger(AaaApplication.class);

    public static void main(String[] args) throws Exception {
        long pid = ProcessHandle.current().pid();
        LOGGER.info("Pid is {}", pid);

        SpringApplication.run(AaaApplication.class, args);
    }
}
