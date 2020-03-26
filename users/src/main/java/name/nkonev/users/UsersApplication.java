package name.nkonev.users;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.sleuth.instrument.messaging.TraceMessagingAutoConfiguration;
import org.springframework.cloud.sleuth.instrument.messaging.TraceSpringIntegrationAutoConfiguration;
import reactor.tools.agent.ReactorDebugAgent;

@SpringBootApplication(exclude = {TraceMessagingAutoConfiguration.class, TraceSpringIntegrationAutoConfiguration.class})
public class UsersApplication {

	public static void main(String[] args) {
		// https://projectreactor.io/docs/core/release/reference/#reactor-tools-debug
		ReactorDebugAgent.init();

		SpringApplication.run(UsersApplication.class, args);
	}

}
