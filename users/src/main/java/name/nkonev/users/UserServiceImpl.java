package name.nkonev.users;

import io.grpc.stub.StreamObserver;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class UserServiceImpl extends UserServiceGrpc.UserServiceImplBase {
    @Override
    public void findByUsername(UserDetailsRequest request, StreamObserver<UserDetailsResponse> responseObserver) {

        UserDetailsResponse response = UserDetailsResponse.newBuilder()
                .setUsername(request.getUsername())
                .setPassword("pw")
                .addAllRoles(List.of("ADMIN", "USER"))
                .build();

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
