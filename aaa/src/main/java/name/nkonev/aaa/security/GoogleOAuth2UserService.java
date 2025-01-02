package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserRequest;
import org.springframework.security.oauth2.client.oidc.userinfo.OidcUserService;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.oidc.user.OidcUser;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;

import java.util.Map;
import java.util.Optional;
import java.util.Set;

import static name.nkonev.aaa.Constants.GOOGLE_LOGIN_PREFIX;

@Service
public class GoogleOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OidcUserRequest, OidcUser> {

    private static final Logger LOGGER = LoggerFactory.getLogger(GoogleOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private OidcUserService oidcUserService;

    @Autowired
    private AaaProperties aaaProperties;

    @Override
    public OidcUser loadUser(OidcUserRequest userRequest) throws OAuth2AuthenticationException {
        OidcUser oAuth2User = oidcUserService.loadUser(userRequest);

        var map = oAuth2User.getAttributes();

        UserAccountDetailsDTO processUserResponse = process(map, userRequest);

        return processUserResponse;
    }


    private String getAvatarUrl(Map<String, Object> map){
        return (String) map.get("picture");
    }

    @Override
    public String getLogin(Map<String, Object> map) {
        String login = (String) map.get("name");
        Assert.hasLength(login, "google name cannot be null");
        login = login.trim();
        login = login.replaceAll(" +", " ");
        return login;
    }

    @Override
    public String getId(Map<String, Object> map) {
        return (String) map.get("sub");
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOAuth2Name() {
        return OAuth2Providers.GOOGLE;
    }

    @Override
    protected Optional<UserAccount> findByOAuth2Id(String oauthId) {
        return userAccountRepository.findByGoogleId(oauthId);
    }

    @Override
    protected UserAccountDetailsDTO setOAuth2IdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        return principal.withOauth2Identifiers(principal.getOauth2Identifiers().withGoogleId(oauthId));
    }

    @Override
    protected UserAccount setOAuth2IdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount = userAccount.withOauthIdentifiers(userAccount.oauth2Identifiers().withGoogleId(oauthId));
        userAccount = userAccountRepository.save(userAccount);
        return userAccount;
    }

    @Override
    protected UserAccount buildEntity(String oauthId, String login, Map<String, Object> map, Set<String> roles) {
        String maybeImageUrl = getAvatarUrl(map);
        UserAccount userAccount = userAccountConverter.buildUserAccountEntityForGoogleInsert(oauthId, login, maybeImageUrl);
        LOGGER.info("Built {} user id={} login='{}'", getOAuth2Name(), oauthId, login);
        return userAccount;
    }

    @Override
    protected String getConflictPrefix() {
        return GOOGLE_LOGIN_PREFIX;
    }

    @Override
    protected ConflictResolveStrategy getConflictResolveStrategy() {
        return aaaProperties.google().resolveConflictsStrategy();
    }

}
