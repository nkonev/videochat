package name.nkonev.aaa.security;

import jakarta.servlet.http.HttpServletRequest;
import name.nkonev.aaa.services.RefererService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.core.endpoint.OAuth2AuthorizationRequest;
import org.springframework.util.StringUtils;

import static name.nkonev.aaa.utils.ServletUtils.getCurrentHttpRequest;

class WithRefererInStateOAuth2AuthorizationRequestResolver implements OAuth2AuthorizationRequestResolver {

    private final DefaultOAuth2AuthorizationRequestResolver delegate;
    private final RefererService referrerService;

    private static final Logger LOGGER = LoggerFactory.getLogger(WithRefererInStateOAuth2AuthorizationRequestResolver.class);

    public WithRefererInStateOAuth2AuthorizationRequestResolver(DefaultOAuth2AuthorizationRequestResolver delegate, RefererService referrerService) {
        this.delegate = delegate;
        this.referrerService = referrerService;
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request);
        return patchState(oAuth2AuthorizationRequest);
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request, String clientRegistrationId) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request, clientRegistrationId);
        return patchState(oAuth2AuthorizationRequest);
    }

    private OAuth2AuthorizationRequest patchState(OAuth2AuthorizationRequest auth2AuthorizationRequest) {
        if (auth2AuthorizationRequest == null) {
            return null;
        }
        OAuth2AuthorizationRequest patched = OAuth2AuthorizationRequest.from(auth2AuthorizationRequest).state(auth2AuthorizationRequest.getState()+getSeparatorRefererOrEmpty()).build();
        return patched;
    }

    private String getSeparatorRefererOrEmpty() {
        HttpServletRequest currentHttpRequest = getCurrentHttpRequest();

        String referer = referrerService.getRefererOrEmpty(currentHttpRequest);
        if (StringUtils.hasLength(referer)){
            LOGGER.info("Storing referrer url {} for still non-user with addr {}", referer, currentHttpRequest.getHeader("x-real-ip"));
            return OAuth2AuthenticationSuccessHandler.SEPARATOR+referer;
        }

        return "";
    }

}
