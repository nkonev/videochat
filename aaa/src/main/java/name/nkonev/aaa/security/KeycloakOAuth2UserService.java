package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.nimbusds.jwt.JWTParser;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.userinfo.DefaultOAuth2UserService;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;
import org.springframework.util.Assert;

import java.text.ParseException;
import java.util.*;
import java.util.stream.Collectors;

import static name.nkonev.aaa.utils.TimeUtil.getNowUTC;


@Component
public class KeycloakOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    private static final Logger LOGGER = LoggerFactory.getLogger(KeycloakOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    public static final String LOGIN_PREFIX = OAuth2Providers.KEYCLOAK + "_";

    @Autowired
    private DefaultOAuth2UserService delegate;

    @Autowired
    private AaaProperties aaaProperties;

    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        OAuth2User oAuth2User = delegate.loadUser(userRequest);

        var map = oAuth2User.getAttributes();

        UserAccountDetailsDTO processUserResponse = process(map, userRequest);

        return processUserResponse;
    }


    private String getAvatarUrl(Map<String, Object> map){
        return null;
    }

    @Override
    protected String getLogin(Map<String, Object> map) {
        String login = (String) map.get("preferred_username");
        Assert.hasLength(login, "keycloak name cannot be null");
        login = login.trim();
        return login;
    }

    @Override
    protected String getId(Map<String, Object> map) {
        return (String) map.get("sub");
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOAuth2Name() {
        return OAuth2Providers.KEYCLOAK;
    }

    @Override
    protected Optional<UserAccount> findByOAuth2Id(String oauthId) {
        return userAccountRepository.findByKeycloakId(oauthId);
    }

    @Override
    protected UserAccountDetailsDTO setOAuth2IdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        return principal.withOauth2Identifiers(principal.getOauth2Identifiers().withKeycloakId(oauthId));
    }

    @Override
    protected UserAccount setOAuth2IdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount = userAccount.withOauthIdentifiers(userAccount.oauth2Identifiers().withKeycloakId(oauthId));
        userAccount = userAccountRepository.save(userAccount);
        return userAccount;
    }

    @Override
    protected UserAccount insertEntity(String oauthId, String login, Map<String, Object> map, Set<String> roles) {
        String maybeImageUrl = getAvatarUrl(map);

        var mappedRoles = RoleMapper.map(aaaProperties.roleMappings().keycloak(), roles);

        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForKeycloakInsert(oauthId, login, maybeImageUrl, mappedRoles, null, false, true, getNowUTC());
        userAccount = userAccountRepository.save(userAccount);
        LOGGER.info("Created {} user id={} login='{}'", getOAuth2Name(), oauthId, login);

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

    @Override
    protected Set<String> getRoles(OAuth2UserRequest userRequest) {
        Set<String> roles = new HashSet<>();
        if (aaaProperties.schedulers().syncKeycloak().syncRoles()) {
            try {
                roles = ((ArrayList<String>) (JWTParser.parse(userRequest.getAccessToken().getTokenValue())).getJWTClaimsSet().getJSONObjectClaim("realm_access").get("roles")).stream().collect(Collectors.toSet());
            } catch (ParseException e) {
                LOGGER.error("Unable to parse roles", e);
            }
        }
        return roles;
    }
}
