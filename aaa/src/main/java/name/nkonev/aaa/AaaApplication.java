package name.nkonev.aaa;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

@EnableAsync
@SpringBootApplication(
        scanBasePackages = {"name.nkonev.aaa"}
)
public class AaaApplication {

    public static void main(String[] args) throws Exception {
        Class<?> cl = Class.forName("org.springframework.security.web.ObservationFilterChainDecorator", true, AaaApplication.class.getClassLoader());
        System.out.println(cl);

        SpringApplication.run(AaaApplication.class, args);
    }
}
