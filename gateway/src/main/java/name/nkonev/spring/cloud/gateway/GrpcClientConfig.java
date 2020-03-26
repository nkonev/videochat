package name.nkonev.spring.cloud.gateway;

import brave.Tracing;
import brave.grpc.GrpcTracing;
import io.grpc.ClientInterceptor;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import name.nkonev.users.UserServiceGrpc;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

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

    @Autowired
    List<ClientInterceptor> clientInterceptorList;


    @Bean(destroyMethod = "shutdown")
    public ManagedChannel grpcClient(GrpcClientProperties properties) {
        // https://codenotfound.com/grpc-java-example.html
        // https://www.baeldung.com/grpc-introduction
        ManagedChannel grpcClient = ManagedChannelBuilder.forAddress(properties.getHost(), properties.getPort())
                .usePlaintext()
                .intercept(clientInterceptorList)
                .build();
        return grpcClient;
    }

    @Bean
    public UserServiceGrpc.UserServiceBlockingStub userService(ManagedChannel grpcClient) {
        return UserServiceGrpc.newBlockingStub(grpcClient);
    }

    @Bean
    public GrpcTracing grpcTracing(Tracing tracing) {
        return GrpcTracing.create(tracing);
    }

    //We also create a client-side interceptor and put that in the context, this interceptor can then be injected into gRPC clients and
    //then applied to the managed channel.
    @Bean
    ClientInterceptor grpcClientSleuthInterceptor(GrpcTracing grpcTracing) {
        return grpcTracing.newClientInterceptor();
    }

}
