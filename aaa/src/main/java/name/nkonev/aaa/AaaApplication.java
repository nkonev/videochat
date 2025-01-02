package name.nkonev.aaa;

import name.nkonev.aaa.config.CleanLogApplicationListener;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.scheduling.annotation.EnableAsync;

@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"name.nkonev.aaa"}
)
public class AaaApplication {

    public static void main(String[] args) throws Exception {
        var builder = new SpringApplicationBuilder(AaaApplication.class);
        builder.listeners(new CleanLogApplicationListener());
        builder.application().run(args);
    }
}
