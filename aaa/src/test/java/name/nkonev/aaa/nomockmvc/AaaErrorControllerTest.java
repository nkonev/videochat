package name.nkonev.aaa.nomockmvc;

import name.nkonev.aaa.AbstractHtmlUnitRunner;
import name.nkonev.aaa.Constants;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;

import java.net.URI;

import static name.nkonev.aaa.controllers.TracerHeaderWriteFilter.EXTERNAL_TRACE_ID_HEADER;
import static org.springframework.http.HttpHeaders.ACCEPT;

public class AaaErrorControllerTest extends AbstractHtmlUnitRunner {

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

}
