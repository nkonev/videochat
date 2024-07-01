package name.nkonev.aaa.security;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.web.authentication.SimpleUrlAuthenticationSuccessHandler;
import org.springframework.util.MultiValueMap;
import org.springframework.web.util.UriComponents;
import org.springframework.web.util.UriComponentsBuilder;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;

public class OAuth2AuthenticationSuccessHandler extends SimpleUrlAuthenticationSuccessHandler {

    public static final String DEFAULT = "/";
    public static final String SEPARATOR = ",";

    private static final Logger LOGGER = LoggerFactory.getLogger(OAuth2AuthenticationSuccessHandler.class);

    @Override
    protected String determineTargetUrl(HttpServletRequest request,
                                        HttpServletResponse response) {

        UriComponents uriComponents = UriComponentsBuilder.newInstance()
                .query(request.getQueryString())
                .build();

        MultiValueMap<String, String> queryParams = uriComponents.getQueryParams();
        String stateEncoded = queryParams.getFirst("state");
        if (stateEncoded == null) {
            return DEFAULT;
        }
        String stateDecoded = URLDecoder.decode(stateEncoded, StandardCharsets.UTF_8);
        String[] split = stateDecoded.split(SEPARATOR);
        if (split.length != 2){
            return DEFAULT;
        } else {
            var url = split[1];
            LOGGER.info("Redirecting user with id {} with addr {} to the restored referrer url {}", SecurityUtils.getPrincipal().getId(), request.getRemoteHost(), url);
            return url;
        }
    }
}
