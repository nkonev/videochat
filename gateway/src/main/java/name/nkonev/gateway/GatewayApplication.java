package name.nkonev.gateway;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.RestController;
import reactor.tools.agent.ReactorDebugAgent;

@SpringBootApplication
@RestController
public class GatewayApplication {

    public static void main(String[] args) {
        // https://projectreactor.io/docs/core/release/reference/#reactor-tools-debug
        ReactorDebugAgent.init();
        SpringApplication.run(GatewayApplication.class, args);
    }

}