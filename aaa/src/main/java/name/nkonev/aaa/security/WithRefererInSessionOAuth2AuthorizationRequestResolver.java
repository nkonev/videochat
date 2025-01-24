package name.nkonev.aaa.security;

import jakarta.servlet.http.HttpServletRequest;
import name.nkonev.aaa.services.RefererService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.oauth2.client.web.DefaultOAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.client.web.OAuth2AuthorizationRequestResolver;
import org.springframework.security.oauth2.core.endpoint.OAuth2AuthorizationRequest;
import org.springframework.util.StringUtils;

import static name.nkonev.aaa.security.OAuth2AuthenticationSuccessHandler.SESSION_ATTR_REDIRECT_URL;

class WithRefererInSessionOAuth2AuthorizationRequestResolver implements OAuth2AuthorizationRequestResolver {

    private final DefaultOAuth2AuthorizationRequestResolver delegate;
    private final RefererService refererService;

    private static final Logger LOGGER = LoggerFactory.getLogger(WithRefererInSessionOAuth2AuthorizationRequestResolver.class);

    public WithRefererInSessionOAuth2AuthorizationRequestResolver(DefaultOAuth2AuthorizationRequestResolver delegate, RefererService refererService) {
        this.delegate = delegate;
        this.refererService = refererService;
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request);
        return saveRedirectUrlIfNeed(request, oAuth2AuthorizationRequest);
    }

    @Override
    public OAuth2AuthorizationRequest resolve(HttpServletRequest request, String clientRegistrationId) {
        OAuth2AuthorizationRequest oAuth2AuthorizationRequest = delegate.resolve(request, clientRegistrationId);
        return saveRedirectUrlIfNeed(request, oAuth2AuthorizationRequest);
    }

    private OAuth2AuthorizationRequest saveRedirectUrlIfNeed(HttpServletRequest request, OAuth2AuthorizationRequest auth2AuthorizationRequest) {
        if (auth2AuthorizationRequest == null) {
            return null;
        }

        var referer = refererService.getRefererOrEmpty(request);
        if (StringUtils.hasLength(referer)){
            var session = request.getSession(true); // true because just after logout there is no session
            if (session != null) {
                LOGGER.info("Storing referer url {} for still non-user with addr {}", referer, request.getHeader("x-real-ip"));
                session.setAttribute(SESSION_ATTR_REDIRECT_URL, referer);
            }
        }

        return auth2AuthorizationRequest;
    }

}
