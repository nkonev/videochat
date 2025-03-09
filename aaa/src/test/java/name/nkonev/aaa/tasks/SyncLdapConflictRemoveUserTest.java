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

import java.time.LocalDateTime;
import java.time.Month;

import static name.nkonev.aaa.TestConstants.*;

@TestPropertySource(properties = {"custom.ldap.resolve-conflicts-strategy=WRITE_NEW_AND_REMOVE_OLD"})
public class SyncLdapConflictRemoveUserTest extends AbstractMockMvcTestRunner {

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
    }

    @Test
    public void syncUpdateFromLdapConflict() {
        var ldapLogin = USER_BEN_LDAP;
        var oldLdapEmail = ldapLogin+"@example.com";

        var ldt = LocalDateTime.of(2000, Month.APRIL, 1, 23, 0, 0);

        UserAccount ldapUserAccount = new UserAccount(
                null,
                CreationType.LDAP,
                ldapLogin, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, oldLdapEmail, null, null, null, null, null, USER_BEN_LDAP_ID, null, ldt, null, null, ldt);
        userAccountRepository.save(ldapUserAccount);

        var ldapUsersBefore = userAccountRepository.countLdap();
        Assertions.assertEquals(1L, ldapUsersBefore);

        // somehow conflicting user with the actual ldap email has appeared
        var nonConflictingLogin = "bbeenn";
        var conflictingEmail = USER_BEN_LDAP_EMAIL;
        UserAccount conflictingUserAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                nonConflictingLogin, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, conflictingEmail, null, null, null, null, null, null, null, null, null, null, null);
        userAccountRepository.save(conflictingUserAccount);

        syncLdapTask.doWork();

        var ldapUsersAfter = userAccountRepository.countLdap();
        Assertions.assertEquals(4L, ldapUsersAfter);

        Assertions.assertTrue(userAccountRepository.findByLogin(nonConflictingLogin).isEmpty()); // removed because conflicted by email

        var replace = userAccountRepository.findByLogin(ldapLogin);
        Assertions.assertTrue(replace.isPresent());
        Assertions.assertEquals(USER_BEN_LDAP_EMAIL, replace.get().email());
    }
}
