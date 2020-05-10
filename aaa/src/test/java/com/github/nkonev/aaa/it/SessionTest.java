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


    private static class SessionHolder {
        final long userId;
        final List<String> sessionCookies;
        String newXsrf;

        public SessionHolder(long userId, List<String> sessionCookies, String newXsrf) {
            this.userId = userId;
            this.sessionCookies = sessionCookies;
            this.newXsrf = newXsrf;
        }

        public SessionHolder(long userId, ResponseEntity responseEntity) {
            this.userId = userId;
            this.sessionCookies = getSessionCookies(responseEntity);
            this.newXsrf = getXsrfValue(getXsrfCookieHeaderValue(responseEntity));
        }

        public String[] getCookiesArray(){
            return sessionCookies.toArray(new String[sessionCookies.size()]);
        }

        public void updateXsrf(ResponseEntity responseEntity){
            this.newXsrf = getXsrfValue(getXsrfCookieHeaderValue(responseEntity));
        }
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

    private SessionHolder login(String login, String password) throws URISyntaxException {
        ResponseEntity<SuccessfulLoginDTO> loginResponseEntity = rawLogin(login, password);

        Assertions.assertEquals(200, loginResponseEntity.getStatusCodeValue());

        return new SessionHolder(loginResponseEntity.getBody().getId(), loginResponseEntity);
    }

    private ResponseEntity<SuccessfulLoginDTO> rawLogin(String login, String password) throws URISyntaxException {
        ResponseEntity<String> getXsrfTokenResponse = testRestTemplate.getForEntity(urlWithContextPath(), String.class);
        String xsrfCookieHeaderValue = getXsrfCookieHeaderValue(getXsrfTokenResponse);
        String xsrf = getXsrfValue(xsrfCookieHeaderValue);


        MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
        params.add(USERNAME_PARAMETER, login);
        params.add(PASSWORD_PARAMETER, password);

        RequestEntity loginRequest = RequestEntity
                .post(new URI(urlWithContextPath()+API_LOGIN_URL))
                .header(HEADER_XSRF_TOKEN, xsrf)
                .header(COOKIE, xsrfCookieHeaderValue)
                .header(ACCEPT, MediaType.APPLICATION_JSON_UTF8_VALUE)
                .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                .body(params);

        return testRestTemplate.exchange(loginRequest, SuccessfulLoginDTO.class);
    }

    private static List<String> getSessionCookies(ResponseEntity<String> loginResponseEntity) {
        return getSetCookieHeaders(loginResponseEntity).stream().dropWhile(s -> s.contains(COOKIE_XSRF+"=;")).collect(Collectors.toList());
    }

    private static String getXsrfValue(String xsrfCookieHeaderValue) {
        return HttpCookie.parse(xsrfCookieHeaderValue).stream().findFirst().orElseThrow(()-> new RuntimeException("cannot get cookie value")).getValue();
    }

    private static String getXsrfCookieHeaderValue(ResponseEntity<String> getXsrfTokenResponse) {
        return getSetCookieHeaders(getXsrfTokenResponse)
                .stream().filter(s -> s.matches(COOKIE_XSRF+"=\\w+.*")).findFirst().orElseThrow(()-> new RuntimeException("cookie " + COOKIE_XSRF + " not found"));
    }

    private static List<String> getSetCookieHeaders(ResponseEntity<String> getXsrfTokenResponse) {
        return Optional.ofNullable(getXsrfTokenResponse.getHeaders().get(HEADER_SET_COOKIE)).orElseThrow(()->new RuntimeException("missed header "+ HEADER_SET_COOKIE));
    }

}
