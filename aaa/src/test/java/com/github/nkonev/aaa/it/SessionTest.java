package name.nkonev.aaa.it;

import name.nkonev.aaa.TestConstants;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.dto.LockDTO;
import name.nkonev.aaa.dto.SuccessfulLoginDTO;
import name.nkonev.aaa.util.ContextPathHelper;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

import java.net.URI;

import static name.nkonev.aaa.TestConstants.*;
import static name.nkonev.aaa.Constants.Urls.PUBLIC_API;
import static name.nkonev.aaa.Constants.Urls.LOCK;
import static org.springframework.http.HttpHeaders.COOKIE;

@DisplayName("Sessions")
public class SessionTest extends OAuth2EmulatorTests {


    @Autowired
    protected AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer;

    public String urlWithContextPath(){
        return ContextPathHelper.urlWithContextPath(abstractConfigurableEmbeddedServletContainer);
    }


    // This test won't works if you call .with(csrf()) before.
    @DisplayName("user cannot request their profile after being locked")
    @Test
    public void userCannotRequestProfileAfterLock() throws Exception {
        SessionHolder userAliceSession = login(TestConstants.USER_LOCKED, TestConstants.COMMON_PASSWORD);
        RequestEntity aliceProfileRequest1 = RequestEntity
                .get(new URI(urlWithContextPath()+ PUBLIC_API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(aliceProfileRequest1, String.class);
        Assertions.assertEquals(200, myPostsResponse1.getStatusCodeValue());


        SessionHolder userAdminSession = login(username, password);
        LockDTO lockDTO = new LockDTO(userAliceSession.userId, true);
        RequestEntity lockRequest = RequestEntity
                .post(new URI(urlWithContextPath()+ PUBLIC_API + Constants.Urls.USER+LOCK))
                .header(HEADER_XSRF_TOKEN, userAdminSession.newXsrf)
                .header(COOKIE, userAdminSession.getCookiesArray())
                .contentType(MediaType.APPLICATION_JSON_UTF8)
                .body(lockDTO);
        ResponseEntity<String> lockResponseEntity = testRestTemplate.exchange(lockRequest, String.class);
        String str = lockResponseEntity.getBody();
        Assertions.assertEquals(200, lockResponseEntity.getStatusCodeValue());


        RequestEntity aliceProfileRequest3 = RequestEntity
                .get(new URI(urlWithContextPath()+ PUBLIC_API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse3 = testRestTemplate.exchange(aliceProfileRequest3, String.class);
        Assertions.assertEquals(401, myPostsResponse3.getStatusCodeValue());

        var newAliceLogin = rawLogin(TestConstants.USER_LOCKED, TestConstants.COMMON_PASSWORD);
        Assertions.assertEquals(401, newAliceLogin.dto().getStatusCodeValue());
    }

}
