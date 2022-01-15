package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import com.github.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Scope;
import org.springframework.context.annotation.ScopedProxyMode;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserRequest;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserService;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;

import java.util.Collection;
import java.util.Map;
import java.util.Optional;
import java.util.Set;


@Transactional
@Scope(proxyMode = ScopedProxyMode.TARGET_CLASS)
@Component
public class GoogleOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OidcUserRequest, OidcUser> {

    private static final Logger LOGGER = LoggerFactory.getLogger(GoogleOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    public static final String LOGIN_PREFIX = OAuth2Providers.GOOGLE + "_";

    @Autowired
    private AaaPreAuthenticationChecks aaaPreAuthenticationChecks;

    @Autowired
    private AaaPostAuthenticationChecks aaaPostAuthenticationChecks;

    @Autowired
    private OidcUserService oidcUserService;

    @Override
    public OidcUser loadUser(OidcUserRequest userRequest) throws OAuth2AuthenticationException {
        OidcUser oAuth2User = oidcUserService.loadUser(userRequest);

        var map = oAuth2User.getAttributes();
        String googleId = getId(map);
        Assert.notNull(googleId, "googleId cannot be null");

        UserAccountDetailsDTO resultPrincipal = mergeOauthIdToExistsUser(googleId);
        if (resultPrincipal != null) {
            // ok
        } else {
            String login = getLogin(map);
            resultPrincipal = createOrGetExistsUser(googleId, login, map, null);
        }

        aaaPreAuthenticationChecks.check(resultPrincipal);
        aaaPostAuthenticationChecks.check(resultPrincipal);
        return resultPrincipal;
    }


    private String getAvatarUrl(Map<String, Object> map){
        return (String) map.get("picture");
    }

    private String getLogin(Map<String, Object> map) {
        String login = (String) map.get("name");
        Assert.hasLength(login, "google name cannot be null");
        login = login.trim();
        login = login.replaceAll(" +", " ");
        return login;
    }

    private String getId(Map<String, Object> map) {
        return (String) map.get("sub");
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOauthName() {
        return OAuth2Providers.GOOGLE;
    }

    @Override
    protected Optional<UserAccount> findByOauthId(String oauthId) {
        return userAccountRepository.findByOauth2IdentifiersGoogleId(oauthId);
    }

    @Override
    protected UserAccountDetailsDTO setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        return principal.withOauth2Identifiers(principal.getOauth2Identifiers().withGoogleId(oauthId));
    }

    @Override
    protected void setOauthIdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount = userAccount.withOauthIdentifiers(userAccount.oauth2Identifiers().withGoogleId(oauthId));
        userAccount = userAccountRepository.save(userAccount);
    }

    @Override
    protected UserAccount insertEntity(String oauthId, String login, Map<String, Object> map, Set<String> roles) {
        String maybeImageUrl = getAvatarUrl(map);
        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForGoogleInsert(oauthId, login, maybeImageUrl);
        userAccount = userAccountRepository.save(userAccount);
        LOGGER.info("Created {} user id={} login='{}'", getOauthName(), oauthId, login);

        return userAccount;
    }

    @Override
    protected String getLoginPrefix() {
        return LOGIN_PREFIX;
    }

    @Override
    protected Optional<UserAccount> findByUsername(String login) {
        return userAccountRepository.findByUsername(login);
    }

}
