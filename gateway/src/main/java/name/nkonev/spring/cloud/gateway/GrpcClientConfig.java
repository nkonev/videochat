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

    @Bean
    public HelloServiceGrpc.HelloServiceBlockingStub conf(GrpcClientProperties properties) {
        // https://codenotfound.com/grpc-java-example.html
        // https://www.baeldung.com/grpc-introduction
        ManagedChannel grpcClient = ManagedChannelBuilder.forAddress(properties.getHost(), properties.getPort())
//                .intercept(grpcTracing.newClientInterceptor())
                .usePlaintext()
                .build();
        HelloServiceGrpc.HelloServiceBlockingStub helloServiceStub = HelloServiceGrpc.newBlockingStub(grpcClient);
        return helloServiceStub;
    }
}
