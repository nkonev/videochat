package name.nkonev.users;

import io.grpc.stub.StreamObserver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import java.util.List;

@Service
public class UserServiceImpl extends UserServiceGrpc.UserServiceImplBase {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserServiceImpl.class);

    @Override
    public void findByUsername(UserDetailsRequest request, StreamObserver<UserDetailsResponse> responseObserver) {

        UserDetailsResponse response = UserDetailsResponse.newBuilder()
                .setUsername(request.getUsername())
                .setPassword("pw")
                .addAllRoles(List.of("ADMIN", "USER"))
                .build();
        LOGGER.info("Responding UserDetails for '{}'", request.getUsername());

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
