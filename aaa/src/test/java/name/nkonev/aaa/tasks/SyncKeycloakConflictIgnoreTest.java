package name.nkonev.aaa.tasks;

import name.nkonev.aaa.AbstractMockMvcTestRunner;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.CreationType;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.context.TestPropertySource;
import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.keycloakEmail;
import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.keycloakLogin;

@TestPropertySource(properties = {"custom.keycloak.resolve-conflicts-strategy=IGNORE"})
public class SyncKeycloakConflictIgnoreTest extends AbstractMockMvcTestRunner {
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private SyncKeycloakTask syncKeycloakTask;

    @Test
    public void syncInsertFromKeycloakConflict() {
        var conflictingLogin = keycloakLogin;
        var nonConflictingEmail = conflictingLogin+"@example1.com";
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                conflictingLogin, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, nonConflictingEmail, null, null, null, null, null, null, null, null, null, null, null);
        userAccountRepository.save(userAccount);
        var before = userAccountRepository.findByLogin(conflictingLogin).get();
        Assertions.assertEquals(nonConflictingEmail, before.email());

        syncKeycloakTask.doWork();

        var after = userAccountRepository.findByLogin(conflictingLogin).get();
        Assertions.assertEquals(nonConflictingEmail, after.email());
        Assertions.assertNotEquals(keycloakEmail, after.email());
        Assertions.assertEquals(before.id(), after.id());
    }

}
