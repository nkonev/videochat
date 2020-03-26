package name.nkonev.spring.cloud.gateway;

import com.google.protobuf.ProtocolStringList;
import name.nkonev.users.UserDetailsRequest;
import name.nkonev.users.UserDetailsResponse;
import name.nkonev.users.UserServiceGrpc;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.userdetails.ReactiveUserDetailsService;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.crypto.factory.PasswordEncoderFactories;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;

@Service
public class GrpcUserDetailsService implements ReactiveUserDetailsService {
    @Autowired
    private UserServiceGrpc.UserServiceBlockingStub userServiceStub;

    private static final Logger LOGGER = LoggerFactory.getLogger(GrpcUserDetailsService.class);

    // TODO check thread safety
    private static final PasswordEncoder encoder = PasswordEncoderFactories.createDelegatingPasswordEncoder();

    @Override
    public Mono<UserDetails> findByUsername(String username) {
        LOGGER.info("Requesting UserDetails for '{}'", username);
        UserDetailsRequest build = UserDetailsRequest.newBuilder().setUsername(username).build();
        return Mono.<UserDetails>defer(()-> {
            LOGGER.debug("In deferred requesting '{}'", username);
            UserDetailsResponse byUsername = userServiceStub.findByUsername(build);
            ProtocolStringList rolesList = byUsername.getRolesList();
            String[] roles = rolesList.toArray(new String[0]);
            // TODO move encoding to user service
            UserDetails user = User.builder().passwordEncoder(encoder::encode)
                    .username(byUsername.getUsername()).password(byUsername.getPassword()).roles(roles).build();
            return Mono.just(user);
        });
    }
}
