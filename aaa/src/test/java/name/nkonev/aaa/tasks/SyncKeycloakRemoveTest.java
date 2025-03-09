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

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;

@TestPropertySource(properties = {"custom.keycloak.resolve-conflicts-strategy=IGNORE"})
public class SyncKeycloakRemoveTest extends AbstractMockMvcTestRunner {
    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private SyncKeycloakTask syncKeycloakTask;

    @Test
    public void syncKeycloak() {
        var login = "user20";
        var email = "user20@example.com";
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.KEYCLOAK,
                login, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, email, null, null, null, null, "20-123", null, null, null, getNowUTC().minusSeconds(1), null, null);
        userAccountRepository.save(userAccount);
        var before = userAccountRepository.findByLogin(login).get();
        Assertions.assertEquals(email, before.email());

        syncKeycloakTask.doWork();

        Assertions.assertFalse(userAccountRepository.findByLogin(login).isPresent());
    }

}
