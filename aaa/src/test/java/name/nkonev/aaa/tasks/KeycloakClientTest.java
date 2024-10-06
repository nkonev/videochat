package name.nkonev.aaa.tasks;

import name.nkonev.aaa.AbstractMockMvcTestRunner;
import name.nkonev.aaa.services.tasks.KeycloakClient;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

import static name.nkonev.aaa.nomockmvc.OAuth2EmulatorTests.*;

public class KeycloakClientTest extends AbstractMockMvcTestRunner {

    @Autowired
    private KeycloakClient keycloakClient;

    @Test
    public void testGetUsers() {
        var users = keycloakClient.getUsers(10, 0);
        Assertions.assertFalse(users.isEmpty());
        var user = users.stream().filter(u -> u.id().equals(keycloakId)).findFirst().get();
        Assertions.assertEquals(keycloakId, user.id());
        Assertions.assertEquals(keycloakLogin, user.username());
        Assertions.assertEquals("User", user.firstName());
        Assertions.assertEquals("Second", user.lastName());
        Assertions.assertEquals(keycloakEmail, user.email());
        Assertions.assertEquals(true, user.enabled());
    }

    @Test
    public void testGetRoles() {
        var roles = keycloakClient.getRoles(10, 0);
        Assertions.assertFalse(roles.isEmpty());
    }

    @Test
    public void testGetRolesEmpty() {
        var roles = keycloakClient.getRoles(10, 4);
        Assertions.assertTrue(roles.isEmpty());
    }

    @Test
    public void testGetUsersInRole() {
        var users = keycloakClient.getUsersInRole("USER", 10, 0);
        Assertions.assertEquals(1, users.size());
        var user = users.get(0);
        Assertions.assertEquals(keycloakId, user.id());
        Assertions.assertEquals(keycloakLogin, user.username());
        Assertions.assertEquals("User", user.firstName());
        Assertions.assertEquals("Second", user.lastName());
        Assertions.assertEquals(keycloakEmail, user.email());
        Assertions.assertEquals(true, user.enabled());
    }

    @Test
    public void testGetUsersInRoleEmpty() {
        var users = keycloakClient.getUsersInRole("USER", 10, 1);
        Assertions.assertTrue(users.isEmpty());
    }
}