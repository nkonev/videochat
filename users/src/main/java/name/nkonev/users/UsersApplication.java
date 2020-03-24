package name.nkonev.users;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import reactor.tools.agent.ReactorDebugAgent;

@SpringBootApplication
public class UsersApplication {

	public static void main(String[] args) {
		// https://projectreactor.io/docs/core/release/reference/#reactor-tools-debug
		ReactorDebugAgent.init();

		SpringApplication.run(UsersApplication.class, args);
	}

}
