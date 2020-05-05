package com.github.nkonev.blog.services;

import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import io.grpc.stub.StreamObserver;

import java.time.Instant;
import java.util.List;
import java.util.stream.Collectors;

import name.nkonev.users.UserServiceGrpc;
import name.nkonev.users.UserSessionRequest;
import name.nkonev.users.UserSessionResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.session.SessionProperties;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.context.SecurityContextImpl;
import org.springframework.session.Session;
import org.springframework.session.data.redis.RedisIndexedSessionRepository;
import org.springframework.stereotype.Service;

@Service
public class UserSessionService extends UserServiceGrpc.UserServiceImplBase {

    private static final Logger LOGGER = LoggerFactory.getLogger(UserSessionService.class);

    @Autowired
    private RedisIndexedSessionRepository redisIndexedSessionRepository;

    @Autowired
    private SessionProperties sessionProperties;

    @Override
    public void findBySession(UserSessionRequest request, StreamObserver<UserSessionResponse> responseObserver) {

        UserSessionResponse response;

        Session session = redisIndexedSessionRepository.findById(request.getSession());
        if (session != null) {
            Instant plus = session.getCreationTime().plus(sessionProperties.getTimeout());
            long expiresIn = plus.toEpochMilli();

            SecurityContextImpl securityContext = session.getAttribute("SPRING_SECURITY_CONTEXT");
            UserAccountDetailsDTO userAccountDetailsDTO = (UserAccountDetailsDTO) securityContext.getAuthentication().getPrincipal();

            List<String> roles = userAccountDetailsDTO.getRoles().stream()
                    .map(GrantedAuthority::getAuthority).collect(Collectors.toList());
            response = UserSessionResponse.newBuilder()
                    .setExpiresIn(expiresIn)
                    .setUserId(userAccountDetailsDTO.getId())
                    .setUserName(userAccountDetailsDTO.getUsername())
                    .addAllRoles(roles)
                    .build();

        } else {
            response = UserSessionResponse.newBuilder()
                    .setExpiresIn(0L)
                    .setUserId(0L)
                    .setUserName("anonymous")
                    .addAllRoles(List.of("ROLE_ANONYMOUS"))
                    .build();
        }
        LOGGER.info("Responding UserSessionResponse for session '{}'", request.getSession());

        responseObserver.onNext(response);
        responseObserver.onCompleted();
    }
}
