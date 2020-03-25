package name.nkonev.users;

import io.grpc.BindableService;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.IOException;
import java.util.List;

@Configuration
@EnableConfigurationProperties(GrpcServerConfig.GrpcClientProperties.class)
public class GrpcServerConfig {

    private static final Logger LOGGER = LoggerFactory.getLogger(GrpcServerConfig.class);

    @ConfigurationProperties("grpc")
    public static class GrpcClientProperties {
        private int port;

        public int getPort() {
            return port;
        }

        public void setPort(int port) {
            this.port = port;
        }
    }

    @Bean(destroyMethod = "shutdown", initMethod = "start")
    public Server grpcClient(GrpcClientProperties properties, List<BindableService> services) {
        // https://codenotfound.com/grpc-java-example.html
        // https://www.baeldung.com/grpc-introduction

        ServerBuilder<?> serverBuilder = ServerBuilder
                .forPort(properties.getPort());

        for (BindableService service: services) {
            LOGGER.info("Adding {}", service);
            serverBuilder = serverBuilder.addService(service);
        }
        Server server =  serverBuilder.build();
        return server;
    }

}
