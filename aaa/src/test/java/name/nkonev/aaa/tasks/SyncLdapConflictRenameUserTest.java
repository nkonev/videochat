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

import static name.nkonev.aaa.Constants.LDAP_CONFLICT_PREFIX;
import static name.nkonev.aaa.TestConstants.USER_BEN_LDAP;
import static name.nkonev.aaa.TestConstants.USER_BEN_LDAP_EMAIL;

@TestPropertySource(properties = {"custom.ldap.resolve-conflicts-strategy=WRITE_NEW_AND_RENAME_OLD"})
public class SyncLdapConflictRenameUserTest extends AbstractMockMvcTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private SyncLdapTask syncLdapTask;

    @Test
    public void syncInsertFromLdapConflict() {
        var conflictingLogin = USER_BEN_LDAP;
        var nonConflictingEmail = conflictingLogin+"@example.com";
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                conflictingLogin, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, nonConflictingEmail, null, null, null, null, null, null, null, null, null, null, null);
        userAccountRepository.save(userAccount);
        var before = userAccountRepository.findByLogin(conflictingLogin).get();
        Assertions.assertEquals(nonConflictingEmail, before.email());

        var ldapUsersBefore = userAccountRepository.countLdap();
        Assertions.assertEquals(0L, ldapUsersBefore);

        syncLdapTask.doWork();

        var ldapUsersAfter = userAccountRepository.countLdap();
        Assertions.assertEquals(4L, ldapUsersAfter);

        var after = userAccountRepository.findByLogin(conflictingLogin).get();
        Assertions.assertNotEquals(nonConflictingEmail, after.email());
        Assertions.assertEquals(USER_BEN_LDAP_EMAIL, after.email());
        Assertions.assertNotEquals(before.id(), after.id());

        Assertions.assertTrue(userAccountRepository.findByLogin(LDAP_CONFLICT_PREFIX + conflictingLogin).isPresent());
    }

}
