package name.nkonev.spring.cloud.gateway;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.sleuth.instrument.messaging.TraceMessagingAutoConfiguration;
import org.springframework.cloud.sleuth.instrument.messaging.TraceSpringIntegrationAutoConfiguration;
import org.springframework.cloud.sleuth.instrument.rpc.TraceRpcAutoConfiguration;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Mono;
import reactor.tools.agent.ReactorDebugAgent;

@SpringBootApplication(exclude = {TraceMessagingAutoConfiguration.class, TraceSpringIntegrationAutoConfiguration.class, TraceRpcAutoConfiguration.class})
@RestController
public class GatewayApplication {

    public static void main(String[] args) {
        // https://projectreactor.io/docs/core/release/reference/#reactor-tools-debug
        ReactorDebugAgent.init();
        SpringApplication.run(GatewayApplication.class, args);
    }

    @GetMapping("/public/hello")
    public Mono<String> hello() {
        return Mono.just("Hello, Spring!");
    }
}