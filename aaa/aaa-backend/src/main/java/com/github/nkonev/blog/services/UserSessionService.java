package com.github.nkonev.blog.services;

import io.grpc.stub.StreamObserver;
import java.util.List;
import name.nkonev.users.UserServiceGrpc;
import name.nkonev.users.UserSessionRequest;
import name.nkonev.users.UserSessionResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class UserSessionService extends UserServiceGrpc.UserServiceImplBase {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserSessionService.class);

    @Override
    public void findBySession(UserSessionRequest request, StreamObserver<UserSessionResponse> responseObserver) {

      UserSessionResponse response = UserSessionResponse.newBuilder()
                .setExpiresIn(1L)
                .setUserId(1L)
                .setUserName("superuser")
                .addAllRoles(List.of("ADMIN", "USER"))
                .build();
        LOGGER.info("Responding UserSessionResponse for session '{}'", request.getSession());

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
