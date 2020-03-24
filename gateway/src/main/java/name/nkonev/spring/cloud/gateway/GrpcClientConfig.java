package name.nkonev.spring.cloud.gateway;

import brave.Tracing;
import com.codenotfound.grpc.helloworld.HelloServiceGrpc;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@EnableConfigurationProperties(GrpcClientConfig.GrpcClientProperties.class)
public class GrpcClientConfig {

    @ConfigurationProperties("grpc")
    public static class GrpcClientProperties {
        private String host;
        private int port;

        public String getHost() {
            return host;
        }

        public void setHost(String host) {
            this.host = host;
        }

        public int getPort() {
            return port;
        }

        public void setPort(int port) {
            this.port = port;
        }
    }

    @Bean(destroyMethod = "shutdown")
    public ManagedChannel grpcClient(GrpcClientProperties properties) {
        // https://codenotfound.com/grpc-java-example.html
        // https://www.baeldung.com/grpc-introduction
        ManagedChannel grpcClient = ManagedChannelBuilder.forAddress(properties.getHost(), properties.getPort())
                .usePlaintext()
                .build();
        return grpcClient;
    }

    @Bean
    public HelloServiceGrpc.HelloServiceBlockingStub helloService(ManagedChannel grpcClient) {
        HelloServiceGrpc.HelloServiceBlockingStub helloServiceStub = HelloServiceGrpc.newBlockingStub(grpcClient);
        return helloServiceStub;
    }
}
