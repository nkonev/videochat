package com.github.nkonev.aaa.it;

import com.github.nkonev.aaa.CommonTestConstants;
import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.LockDTO;
import com.github.nkonev.aaa.dto.SuccessfulLoginDTO;
import com.github.nkonev.aaa.util.ContextPathHelper;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.web.servlet.server.AbstractServletWebServerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import java.net.HttpCookie;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.CommonTestConstants.*;
import static com.github.nkonev.aaa.Constants.Urls.API;
import static com.github.nkonev.aaa.Constants.Urls.LOCK;
import static com.github.nkonev.aaa.security.SecurityConfig.*;
import static org.springframework.http.HttpHeaders.ACCEPT;
import static org.springframework.http.HttpHeaders.COOKIE;

public class SessionTest extends OAuth2EmulatorTests {


    @Autowired
    protected AbstractServletWebServerFactory abstractConfigurableEmbeddedServletContainer;

    public String urlWithContextPath(){
        return ContextPathHelper.urlWithContextPath(abstractConfigurableEmbeddedServletContainer);
    }


    // This test won't works if you call .with(csrf()) before.
    @Test
    public void userCannotLoginAfterLock() throws Exception {
        SessionHolder userAliceSession = login(CommonTestConstants.USER_LOCKED, CommonTestConstants.COMMON_PASSWORD);

        RequestEntity myPostsRequest1 = RequestEntity
                .get(new URI(urlWithContextPath()+ API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(myPostsRequest1, String.class);
        Assertions.assertEquals(200, myPostsResponse1.getStatusCodeValue());


        SessionHolder userAdminSession = login(username, password);
        LockDTO lockDTO = new LockDTO(userAliceSession.userId, true);
        RequestEntity lockRequest = RequestEntity
                .post(new URI(urlWithContextPath()+API+ Constants.Urls.USER+LOCK))
                .header(HEADER_XSRF_TOKEN, userAdminSession.newXsrf)
                .header(COOKIE, userAdminSession.getCookiesArray())
                .contentType(MediaType.APPLICATION_JSON_UTF8)
                .body(lockDTO);
        ResponseEntity<String> lockResponseEntity = testRestTemplate.exchange(lockRequest, String.class);
        String str = lockResponseEntity.getBody();
        Assertions.assertEquals(200, lockResponseEntity.getStatusCodeValue());


        RequestEntity myPostsRequest3 = RequestEntity
                .get(new URI(urlWithContextPath()+ API + Constants.Urls.PROFILE))
                .header(HEADER_XSRF_TOKEN, userAliceSession.newXsrf)
                .header(COOKIE, userAliceSession.getCookiesArray())
                .build();
        ResponseEntity<String> myPostsResponse3 = testRestTemplate.exchange(myPostsRequest3, String.class);
        Assertions.assertEquals(401, myPostsResponse3.getStatusCodeValue());


        ResponseEntity<SuccessfulLoginDTO> newAliceLogin = rawLogin(CommonTestConstants.USER_LOCKED, CommonTestConstants.COMMON_PASSWORD);
        Assertions.assertEquals(401, newAliceLogin.getStatusCodeValue());
    }

}
