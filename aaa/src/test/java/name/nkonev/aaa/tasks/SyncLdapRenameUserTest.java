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
public class SyncLdapRenameUserTest extends AbstractMockMvcTestRunner {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private SyncLdapTask syncLdapTask;

    @Test
    public void syncLdap() {
        var conflictingLogin = USER_BEN_LDAP;
        var conflictingEmail = conflictingLogin+"@example.com";
        UserAccount userAccount = new UserAccount(
                null,
                CreationType.REGISTRATION,
                conflictingLogin, null, null, null, null,false, false, true, true,
                new UserRole[]{UserRole.ROLE_USER}, conflictingEmail, null, null, null, null, null, null, null, null);
        userAccountRepository.save(userAccount);
        var before = userAccountRepository.findByUsername(conflictingLogin).get();
        Assertions.assertEquals(conflictingEmail, before.email());

        var ldapUsersBefore = jdbcTemplate.queryForObject("select count (*) from user_account where ldap_id is not null", Long.class);
        Assertions.assertEquals(0L, ldapUsersBefore);

        syncLdapTask.doWork();

        var ldapUsersAfter = jdbcTemplate.queryForObject("select count (*) from user_account where ldap_id is not null", Long.class);
        Assertions.assertEquals(4L, ldapUsersAfter);

        var after = userAccountRepository.findByUsername(conflictingLogin).get();
        Assertions.assertNotEquals(conflictingEmail, after.email());
        Assertions.assertEquals(USER_BEN_LDAP_EMAIL, after.email());
        Assertions.assertNotEquals(before.id(), after.id());

        Assertions.assertTrue(userAccountRepository.findByUsername(LDAP_CONFLICT_PREFIX + conflictingLogin).isPresent());
    }

}
