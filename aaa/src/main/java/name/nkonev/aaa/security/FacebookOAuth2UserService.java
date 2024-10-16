package name.nkonev.aaa.security;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.config.properties.ConflictResolveStrategy;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
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

import java.util.Map;
import java.util.Optional;
import java.util.Set;

import static name.nkonev.aaa.Constants.FACEBOOK_LOGIN_PREFIX;


@Component
public class FacebookOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    private static final Logger LOGGER = LoggerFactory.getLogger(FacebookOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

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
        String login = (String) map.get("name");
        Assert.hasLength(login, "facebook name cannot be null");
        login = login.trim();
        login = login.replaceAll(" +", " ");
        return login;
    }

    @Override
    protected String getId(Map<String, Object> map) {
        return (String) map.get("id");
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOAuth2Name() {
        return OAuth2Providers.FACEBOOK;
    }

    @Override
    protected Optional<UserAccount> findByOAuth2Id(String oauthId) {
        return userAccountRepository.findByFacebookId(oauthId);
    }

    @Override
    protected UserAccountDetailsDTO setOAuth2IdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        return principal.withOauth2Identifiers(principal.getOauth2Identifiers().withFacebookId(oauthId));
    }

    @Override
    protected UserAccount setOAuth2IdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount = userAccount.withOauthIdentifiers(userAccount.oauth2Identifiers().withFacebookId(oauthId));
        userAccount = userAccountRepository.save(userAccount);
        return userAccount;
    }

    @Override
    protected UserAccount buildEntity(String oauthId, String login, Map<String, Object> map, Set<String> roles) {
        String maybeImageUrl = getAvatarUrl(map);
        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForFacebookInsert(oauthId, login, maybeImageUrl);
        LOGGER.info("Built {} user id={} login='{}'", getOAuth2Name(), oauthId, login);
        return userAccount;
    }

    @Override
    protected String getConflictPrefix() {
        return FACEBOOK_LOGIN_PREFIX;
    }

    @Override
    protected ConflictResolveStrategy getConflictResolveStrategy() {
        return aaaProperties.facebook().resolveConflictsStrategy();
    }

}
