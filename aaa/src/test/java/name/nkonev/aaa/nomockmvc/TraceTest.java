package name.nkonev.aaa.nomockmvc;

import name.nkonev.aaa.AbstractTestRunner;
import name.nkonev.aaa.Constants;
import name.nkonev.aaa.dto.SuccessfulLoginDTO;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;

import java.net.URI;

import static name.nkonev.aaa.TestConstants.HEADER_XSRF_TOKEN;
import static name.nkonev.aaa.controllers.TracerHeaderWriteFilter.EXTERNAL_TRACE_ID_HEADER;
import static name.nkonev.aaa.security.SecurityConfig.*;
import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.http.HttpHeaders.ACCEPT;
import static org.springframework.http.HttpHeaders.COOKIE;

@ExtendWith(OutputCaptureExtension.class)
public class TraceTest extends AbstractTestRunner {

    @Test
    public void testTraceId() throws Exception {
        RequestEntity myPostsRequest1 = RequestEntity
                .get(new URI(urlWithContextPath()+ Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH))
                .header(ACCEPT, MediaType.TEXT_HTML.toString(), MediaType.ALL.toString())
                .build();
        ResponseEntity<String> myPostsResponse1 = testRestTemplate.exchange(myPostsRequest1, String.class);
        Assertions.assertEquals(401, myPostsResponse1.getStatusCodeValue());

        var traceId = myPostsResponse1.getHeaders().getFirst(EXTERNAL_TRACE_ID_HEADER);
        Assertions.assertFalse(traceId.isEmpty());

        var body = myPostsResponse1.getBody();
        Assertions.assertTrue(body.contains(traceId));
    }

    @Test
    public void testLoginSuccessful(CapturedOutput output) throws Exception {
        // copy-paste of rawLogin
        var xsrfHolder = getXsrf();
        String xsrfCookieHeaderValue = xsrfHolder.xsrfCookieHeaderValue;
        String xsrf = xsrfHolder.newXsrf;

        MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
        params.add(USERNAME_PARAMETER, username);
        params.add(PASSWORD_PARAMETER, password);

        RequestEntity loginRequest = RequestEntity
                .post(new URI(urlWithContextPath()+API_LOGIN_URL))
                .header(HEADER_XSRF_TOKEN, xsrf)
                .header(COOKIE, xsrfCookieHeaderValue)
                .header(ACCEPT, MediaType.APPLICATION_JSON_UTF8_VALUE)
                .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                .body(params);

        ResponseEntity<SuccessfulLoginDTO> loginResponseEntity = testRestTemplate.exchange(loginRequest, SuccessfulLoginDTO.class);

        var traceId = loginResponseEntity.getHeaders().getFirst(EXTERNAL_TRACE_ID_HEADER);
        Assertions.assertFalse(traceId.isEmpty());

        assertThat(output).contains(traceId);
    }
}
