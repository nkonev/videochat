package name.nkonev.users;

import com.codenotfound.grpc.helloworld.HelloRequest;
import com.codenotfound.grpc.helloworld.HelloResponse;
import com.codenotfound.grpc.helloworld.HelloServiceGrpc;
import io.grpc.stub.StreamObserver;
import org.springframework.stereotype.Service;

@Service
public class HelloServiceImpl extends HelloServiceGrpc.HelloServiceImplBase {
    @Override
    public void hello(
            HelloRequest request, StreamObserver<HelloResponse> responseObserver) {

        String greeting = new StringBuilder()
                .append("Hello, ")
                .append(request.getFirstName())
                .append(" ")
                .append(request.getLastName())
                .toString();

        HelloResponse response = HelloResponse.newBuilder()
                .setGreeting(greeting)
                .build();

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
