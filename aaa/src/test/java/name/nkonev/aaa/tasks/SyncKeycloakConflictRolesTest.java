package name.nkonev.aaa.tasks;

import name.nkonev.aaa.AbstractMockMvcTestRunner;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.context.TestPropertySource;

import java.util.Arrays;
import java.util.Set;
import java.util.stream.Collectors;

import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.keycloakEmail;
import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.keycloakLogin;

@TestPropertySource(properties = {"custom.keycloak.resolve-conflicts-strategy=IGNORE"})
public class SyncKeycloakConflictRolesTest extends AbstractMockMvcTestRunner {
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private SyncKeycloakTask syncKeycloakTask;

    @Test
    public void syncKeycloak() {
        var login = keycloakLogin;

        syncKeycloakTask.doWork();

        var after = userAccountRepository.findByUsername(login).get();
        Assertions.assertEquals(Set.of(UserRole.ROLE_USER, UserRole.ROLE_ADMIN), Arrays.stream(after.roles()).collect(Collectors.toSet()));
        Assertions.assertEquals(keycloakLogin, after.username());
        Assertions.assertEquals(keycloakEmail, after.email());
    }

}
