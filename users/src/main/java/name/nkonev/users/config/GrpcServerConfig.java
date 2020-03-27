package name.nkonev.users.config;

import brave.Tracing;
import brave.grpc.GrpcTracing;
import io.grpc.BindableService;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.ServerInterceptor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

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
    public Server grpcClient(GrpcClientProperties properties, List<BindableService> services, ServerInterceptor serverInterceptor) {
        // https://codenotfound.com/grpc-java-example.html
        // https://www.baeldung.com/grpc-introduction

        ServerBuilder<?> serverBuilder = ServerBuilder
                .forPort(properties.getPort());
        for (BindableService service: services) {
            LOGGER.info("Adding {}", service);
            serverBuilder = serverBuilder.addService(service);
        }
        serverBuilder = serverBuilder.intercept(serverInterceptor);
        Server server = serverBuilder.build();
        return server;
    }

    @Bean
    public GrpcTracing grpcTracing(Tracing tracing) {
        return GrpcTracing.create(tracing);
    }

    //grpc-spring-boot-starter provides @GrpcGlobalInterceptor to allow server-side interceptors to be registered with all
    //server stubs, we are just taking advantage of that to install the server-side gRPC tracer.
    @Bean
    ServerInterceptor grpcServerSleuthInterceptor(GrpcTracing grpcTracing) {
        return grpcTracing.newServerInterceptor();
    }

}
