package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import com.github.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import com.nimbusds.jose.shaded.json.JSONArray;
import com.nimbusds.jwt.JWTParser;
import com.nimbusds.jwt.SignedJWT;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Scope;
import org.springframework.context.annotation.ScopedProxyMode;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.oauth2.client.userinfo.DefaultOAuth2UserService;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;

import java.text.ParseException;
import java.util.*;
import java.util.stream.Collectors;


@Transactional
@Scope(proxyMode = ScopedProxyMode.TARGET_CLASS)
@Component
public class KeycloakOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    private static final Logger LOGGER = LoggerFactory.getLogger(KeycloakOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    public static final String LOGIN_PREFIX = OAuth2Providers.KEYCLOAK + "_";

    @Autowired
    private AaaPreAuthenticationChecks aaaPreAuthenticationChecks;

    @Autowired
    private AaaPostAuthenticationChecks aaaPostAuthenticationChecks;

    @Autowired
    private DefaultOAuth2UserService delegate;

    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        OAuth2User oAuth2User = delegate.loadUser(userRequest);

        var map = oAuth2User.getAttributes();
        String keycloakId = getId(map);
        Assert.notNull(keycloakId, "keycloakId cannot be null");

        Set<String> roles = new HashSet<>();
        try {
            roles = ((JSONArray) (JWTParser.parse(userRequest.getAccessToken().getTokenValue())).getJWTClaimsSet().getJSONObjectClaim("realm_access").get("roles")).stream().map(Object::toString).collect(Collectors.toSet());
        } catch (ParseException e) {
            LOGGER.error("Unable to parse roles", e);
        }

        UserAccountDetailsDTO resultPrincipal = mergeOauthIdToExistsUser(keycloakId);
        if (resultPrincipal != null) {
            // ok
        } else {
            String login = getLogin(map);
            userRequest.getAccessToken().getTokenValue();
            resultPrincipal = createOrGetExistsUser(keycloakId, login, map, roles);
        }

        aaaPreAuthenticationChecks.check(resultPrincipal);
        aaaPostAuthenticationChecks.check(resultPrincipal);
        return resultPrincipal;
    }


    private String getAvatarUrl(Map<String, Object> map){
        return null;
    }

    private String getLogin(Map<String, Object> map) {
        String login = (String) map.get("preferred_username");
        Assert.hasLength(login, "keycloak name cannot be null");
        login = login.trim();
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
        return OAuth2Providers.KEYCLOAK;
    }

    @Override
    protected Optional<UserAccount> findByOauthId(String oauthId) {
        return userAccountRepository.findByOauth2IdentifiersKeycloakId(oauthId);
    }

    @Override
    protected UserAccountDetailsDTO setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        return principal.withOauth2Identifiers(principal.getOauth2Identifiers().withKeycloakId(oauthId));
    }

    @Override
    protected void setOauthIdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount = userAccount.withOauthIdentifiers(userAccount.oauth2Identifiers().withKeycloakId(oauthId));
        userAccount = userAccountRepository.save(userAccount);
    }

    @Override
    protected UserAccount insertEntity(String oauthId, String login, Map<String, Object> map, Set<String> roles) {
        String maybeImageUrl = getAvatarUrl(map);
        boolean hasAdminRole = Optional.ofNullable(roles).orElse(new HashSet<>()).stream().anyMatch(s -> "ROLE_ADMIN".equalsIgnoreCase(s) || "ADMIN".equalsIgnoreCase(s));
        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForKeycloakInsert(oauthId, login, maybeImageUrl, hasAdminRole);
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
