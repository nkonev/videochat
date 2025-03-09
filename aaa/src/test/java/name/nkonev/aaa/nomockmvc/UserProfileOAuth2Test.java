package name.nkonev.aaa.nomockmvc;

import name.nkonev.aaa.AbstractHtmlUnitRunner;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.dto.UserRole;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.security.OAuth2Providers;
import org.awaitility.Awaitility;
import org.htmlunit.WebResponse;
import org.htmlunit.html.HtmlButton;
import org.htmlunit.html.HtmlInput;
import org.htmlunit.html.HtmlPage;
import org.htmlunit.util.Cookie;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.password.PasswordEncoder;

import java.io.IOException;
import java.net.URI;
import java.time.Duration;
import java.util.Arrays;

import static name.nkonev.aaa.TestConstants.*;
import static name.nkonev.aaa.Constants.Urls.EXTERNAL_API;
import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;
import static org.springframework.http.HttpHeaders.COOKIE;

@ExtendWith(OutputCaptureExtension.class)
public class UserProfileOAuth2Test extends AbstractHtmlUnitRunner {

    @Autowired
    private PasswordEncoder passwordEncoder;

    private HtmlPage currentPage;

    @BeforeAll
    public static void ba() {
        Awaitility.setDefaultTimeout(Duration.ofSeconds(30));
    }

    private void openOauth2TestPage() throws IOException {
        openOauth2TestPage("");
    }

    private void openOauth2TestPage(String suffix) throws IOException {
        currentPage = webClient.getPage(templateEngineUrlPrefix+"/oauth2.html" + suffix);
    }

    private void clickFacebook() throws IOException {
        currentPage = currentPage.getElementById("a-facebook").click();
    }

    private void clickVkontakte() throws IOException {
        currentPage = currentPage.getElementById("a-vkontakte").click();
    }

    private WebResponse clickVkontakteAndReturn() throws IOException {
        return currentPage.getElementById("a-vkontakte").click().getWebResponse();
    }

    private void clickGoogle() throws IOException {
        currentPage = currentPage.getElementById("a-google").click();
    }

    private void clickKeycloak() throws IOException {
        currentPage = currentPage.getElementById("a-keycloak").click();
    }

    private void clickLogout() throws IOException {
        currentPage.getElementById("btn-logout").click();
    }

    private class LoginPage {
        public LoginPage(String login, String password) {
            this.login = login;
            this.password = password;
        }

        private void openLoginPage() throws IOException {
            currentPage = webClient.getPage(templateEngineUrlPrefix+"/login.html");
        }

        private String login;
        private String password;

        private void login() throws IOException {
            ((HtmlInput)currentPage.getElementById("login")).setValueAttribute(this.login);
            ((HtmlInput)currentPage.getElementById("password")).setValueAttribute(this.password);
            currentPage.getElementById("btn-login").click();
        }
    }

    private class KeycloakLoginPage {
        public KeycloakLoginPage(String login, String password) {
            this.login = login;
            this.password = password;
        }

        private String login;
        private String password;

        private void login() throws IOException {
            ((HtmlInput)currentPage.querySelector("#kc-form-login input#username")).setValueAttribute(this.login);
            ((HtmlInput)currentPage.querySelector("#kc-form-login input#password")).setValueAttribute(this.password);
            currentPage = ((HtmlButton)currentPage.querySelector("#kc-form-login button#kc-login")).click();
        }
    }

    @Test
    public void testFacebookLogin() throws InterruptedException, IOException {

        openOauth2TestPage();

        clickFacebook();

        UserAccount userAccount = await().ignoreExceptions().until(() -> userAccountRepository.findByLogin(facebookLogin).orElseThrow(), o -> true);
        Assertions.assertNotNull(userAccount.id());
        Assertions.assertEquals(facebookLogin, userAccount.login());
    }

    @Test
    public void testFacebookLoginAndAlsoBindVkontakte(CapturedOutput output) throws InterruptedException, IOException {

        openOauth2TestPage("?q=тест");

        clickFacebook();

        await().until(() -> output.getAll().contains("Storing referer url http://localhost:9080/oauth2.html?q=%D1%82%D0%B5%D1%81%D1%82 for still non-user with addr"));

        UserAccount userAccount = await().ignoreExceptions().until(() -> userAccountRepository.findByLogin(facebookLogin).orElseThrow(), o -> true);
        Long facebookLoggedId = userAccount.id();
        Assertions.assertNotNull(facebookLoggedId);
        Assertions.assertEquals(facebookLogin, userAccount.login());
        String facebookId = userAccount.oauth2Identifiers().facebookId();
        Assertions.assertNotNull(facebookId);
        Assertions.assertNull(userAccount.oauth2Identifiers().vkontakteId());
        long count = userAccountRepository.count();

        // here we rely on redirect in WithRefererInStateOAuth2AuthorizationRequestResolver and OAuth2AuthenticationSuccessHandler
        await().until(() -> output.getAll().contains("Redirecting user with id"));
        assertThat(output).contains("to the restored referer url http://localhost:9080/oauth2.html?q=%D1%82%D0%B5%D1%81%D1%82");

        clickVkontakte();

        UserAccount userAccountFbAndVk = await().ignoreExceptions().until(() -> userAccountRepository.findByLogin(facebookLogin).orElseThrow(), o -> true);
        String userAccountFbAndVkFacebookId = userAccountFbAndVk.oauth2Identifiers().facebookId();
        Assertions.assertNotNull(userAccountFbAndVkFacebookId);
        Assertions.assertNotNull(userAccountFbAndVk.oauth2Identifiers().vkontakteId());
        long countAfterVk = userAccountRepository.count();


        Assertions.assertEquals(facebookId, userAccountFbAndVkFacebookId);
        Assertions.assertEquals(count, countAfterVk);
        Assertions.assertEquals(userAccount.login(), userAccountFbAndVk.login());
    }

    @Test
    public void testVkontakteLoginAndDelete() throws Exception {
        final String vkontaktePassword = "dummy password";

        long countInitial = userAccountRepository.count();

        openOauth2TestPage();

        clickVkontakte();

        long countBeforeDelete = await().ignoreExceptions().until(() -> {
            long c = userAccountRepository.count();
            if (countInitial+1 != c) {
                throw new RuntimeException("User is still not created");
            }
            return c;
        }, o -> true);

        UserAccount userAccount = userAccountRepository.findByLogin(vkontakteLogin).orElseThrow();

        Assertions.assertNotNull(userAccount.id());
        Assertions.assertNull(userAccount.avatar());
        Assertions.assertEquals(vkontakteLogin, userAccount.login());

        userAccount = userAccount.withPassword(passwordEncoder.encode(vkontaktePassword));
        userAccountRepository.save(userAccount);


        SessionHolder userNikitaSession = login(vkontakteLogin, vkontaktePassword);
        RequestEntity selfDeleteRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ EXTERNAL_API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userNikitaSession.newXsrf)
                .header(COOKIE, userNikitaSession.getCookiesArray())
                .build();
        ResponseEntity<String> selfDeleteResponse1 = testRestTemplate.exchange(selfDeleteRequest1, String.class);
        Assertions.assertEquals(200, selfDeleteResponse1.getStatusCodeValue());

        long countAfter = await().ignoreExceptions().until(() -> userAccountRepository.count(), o -> true);
        Assertions.assertEquals(countBeforeDelete-1, countAfter);
    }

    @Test
    public void testBindFacebookToTheCurrentAccountThenConflictOnTryingToBindForeignVkontakte(CapturedOutput output) throws Exception {
        long countInitial = userAccountRepository.count();

        // login as regular user 600
        final String login600 = "generated_user_600";
        LoginPage loginPage = new LoginPage(login600, COMMON_PASSWORD);
        loginPage.openLoginPage();
        loginPage.login();
        UserAccount userAccount = userAccountRepository.findByLogin(login600).orElseThrow();
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // bind facebook to him
        openOauth2TestPage();
        clickFacebook();
        Assertions.assertEquals(countInitial, userAccountRepository.count());

        // logout
        clickLogout();

        // check that binding is preserved
        currentPage.refresh();

        // assert that he has facebook id
        UserAccount userAccountAfterBind = userAccountRepository.findByLogin(login600).orElseThrow();
        Assertions.assertNotNull(userAccountAfterBind.oauth2Identifiers().facebookId());

        // login as another user to vk - vk id #1 save to database
        {
            openOauth2TestPage();
            clickVkontakte();

            Assertions.assertEquals(countInitial+1, userAccountRepository.count());
            // logout another user
            clickLogout();
        }

        // login facebook-bound user 600 again
        loginPage.openLoginPage();
        loginPage.login();
        // try to bind him vk, but emulator returns previous vk id #1 - here backend must argue that we already have vk id #1 in our database on another user
        openOauth2TestPage();
        webClient.getOptions().setThrowExceptionOnFailingStatusCode(false);
        final WebResponse vkLoginResponse = clickVkontakteAndReturn();

        Assertions.assertTrue(vkLoginResponse.getContentAsString().contains("Somebody already taken this vkontakte id"));

        assertThat(output).contains("Handling OAuth2IdConflictException, message");
    }

    @Test
    public void checkUnbindFacebook() throws Exception {
        // login as 550
        final String login600 = "generated_user_550";
        LoginPage loginModal600 = new LoginPage(login600, COMMON_PASSWORD);
        loginModal600.openLoginPage();
        loginModal600.login();
        UserAccount userAccount = userAccountRepository.findByLogin(loginModal600.login).orElseThrow();

        // bind facebook
        openOauth2TestPage();
        clickFacebook();

        UserAccount userAccountAfterBindFacebook = userAccountRepository.findByLogin(loginModal600.login).orElseThrow();
        // assert facebook is bound - check database
        Assertions.assertNotNull(userAccountAfterBindFacebook.oauth2Identifiers().facebookId());

        // logout
        clickLogout();

        // login again
        SessionHolder userAliceSession = login(loginModal600.login, loginModal600.password);

        final String FACEBOOK = "/" + OAuth2Providers.FACEBOOK;

        // unbind facebook
        RequestEntity myPostsRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE+FACEBOOK))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(myPostsRequest1, String.class);
        Assertions.assertEquals(200, myPostsResponse1.getStatusCodeValue());

        // assert facebook is unbound - check database
        UserAccount userAccountAfterDeleteFacebook = userAccountRepository.findByLogin(loginModal600.login).orElseThrow();
        Assertions.assertNull(userAccountAfterDeleteFacebook.oauth2Identifiers().facebookId());
    }

    @Test
    public void testGoogleLoginAndSelfDelete() throws Exception {
        final String googlePassword = "dummy password";

        long countInitial = userAccountRepository.count();

        openOauth2TestPage();

        clickGoogle();

        long countBeforeDelete = await().ignoreExceptions().until(() -> {
            long c = userAccountRepository.count();
            if (countInitial+1 != c) {
                throw new RuntimeException("User still not created");
            }
            return c;
        }, o -> true);

        UserAccount userAccount = userAccountRepository.findByLogin(googleLogin).orElseThrow();

        Assertions.assertNotNull(userAccount.id());
        Assertions.assertNull(userAccount.avatar());
        Assertions.assertEquals(googleLogin, userAccount.login());
        Assertions.assertEquals(googleId, userAccount.oauth2Identifiers().googleId());

        userAccount = userAccount.withPassword(passwordEncoder.encode(googlePassword));
        userAccountRepository.save(userAccount);


        SessionHolder userNikitaSession = login(googleLogin, googlePassword);
        RequestEntity selfDeleteRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ EXTERNAL_API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userNikitaSession.newXsrf)
                .header(COOKIE, userNikitaSession.getCookiesArray())
                .build();
        ResponseEntity<String> selfDeleteResponse1 = testRestTemplate.exchange(selfDeleteRequest1, String.class);
        Assertions.assertEquals(200, selfDeleteResponse1.getStatusCodeValue());

        long countAfter = await().ignoreExceptions().until(() -> userAccountRepository.count(), o -> true);
        Assertions.assertEquals(countBeforeDelete-1, countAfter);
    }


    @Test
    public void testKeycloakLoginAndUnbind() throws Exception {
        openOauth2TestPage();

        clickKeycloak();
        KeycloakLoginPage klp = new KeycloakLoginPage(keycloakLogin, keycloakPassword);
        klp.login();

        UserAccount userAccount = userAccountRepository.findByLogin(keycloakLogin).orElseThrow();

        Assertions.assertNotNull(userAccount.id());
        Assertions.assertNull(userAccount.avatar());
        Assertions.assertEquals(keycloakLogin, userAccount.login());
        Assertions.assertEquals(keycloakId, userAccount.oauth2Identifiers().keycloakId());
        Assertions.assertTrue(Arrays.asList(userAccount.roles()).contains(UserRole.ROLE_USER));
        Assertions.assertTrue(Arrays.asList(userAccount.roles()).contains(UserRole.ROLE_ADMIN));

        final String bindDeleteUrl = "/" + OAuth2Providers.KEYCLOAK;

        currentPage = (HtmlPage) currentPage.refresh();
        Cookie xsrf = webClient.getCookieManager().getCookie(COOKIE_XSRF);
        String xsrfValue = xsrf.getValue();
        Cookie session = webClient.getCookieManager().getCookie(getAuthCookieName());

        // unbind keycloak
        RequestEntity myPostsRequest1 = RequestEntity
                .delete(new URI(urlWithContextPath()+ Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE+bindDeleteUrl))
                .header(HEADER_XSRF_TOKEN, xsrfValue)
                .header(COOKIE, session.toString(), xsrf.toString())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(myPostsRequest1, String.class);
        Assertions.assertEquals(200, myPostsResponse1.getStatusCodeValue());

        // assert keycloak is unbound - check database
        UserAccount userAccountAfterDeleteFacebook = userAccountRepository.findByLogin(keycloakLogin).orElseThrow();
        Assertions.assertNull(userAccountAfterDeleteFacebook.oauth2Identifiers().keycloakId());

    }

}
