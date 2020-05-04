package com.github.nkonev.blog.security;

import com.github.nkonev.blog.controllers.ImageUserAvatarUploadController;
import com.github.nkonev.blog.converter.UserAccountConverter;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.security.checks.BlogPostAuthenticationChecks;
import com.github.nkonev.blog.security.checks.BlogPreAuthenticationChecks;
import com.github.nkonev.blog.utils.ImageDownloader;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Scope;
import org.springframework.context.annotation.ScopedProxyMode;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserService;
import org.springframework.security.oauth2.core.OAuth2AuthenticationException;
import org.springframework.security.oauth2.core.user.OAuth2User;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;

import java.util.Map;
import java.util.Optional;



@Transactional
@Scope(proxyMode = ScopedProxyMode.TARGET_CLASS)
@Component
public class FacebookOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    private static final Logger LOGGER = LoggerFactory.getLogger(FacebookOAuth2UserService.class);

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private ImageUserAvatarUploadController imageUserAvatarUploadController;

    @Autowired
    private ImageDownloader imageDownloader;

    public static final String LOGIN_PREFIX = "facebook_";

    @Autowired
    private BlogPreAuthenticationChecks blogPreAuthenticationChecks;

    @Autowired
    private BlogPostAuthenticationChecks blogPostAuthenticationChecks;


    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        OAuth2User oAuth2User = delegate.loadUser(userRequest);

        var map = oAuth2User.getAttributes();
        String facebookId = getId(map);
        Assert.notNull(facebookId, "facebookId cannot be null");


        UserAccountDetailsDTO resultPrincipal = mergeOauthIdToExistsUser(facebookId);
        if (resultPrincipal != null) {
            // ok
        } else {
            String login = getLogin(map);
            resultPrincipal = createOrGetExistsUser(facebookId, login, map);
        }

        blogPreAuthenticationChecks.check(resultPrincipal);
        blogPostAuthenticationChecks.check(resultPrincipal);
        return resultPrincipal;
    }


    private String getAvatarUrl(Map<String, Object> map){
        try {
            String url = (String) ((Map<String, Object>) ((Map<String, Object>) map.get("picture")).get("data")).get("url");
            return imageDownloader.downloadImageAndSave(url, imageUserAvatarUploadController);
        } catch (Exception e){
            LOGGER.info("Cannot get image url from {}, returning null", map);
            return null;
        }
    }

    private String getLogin(Map<String, Object> map) {
        String login = (String) map.get("name");
        Assert.hasLength(login, "facebook name cannot be null");
        login = login.trim();
        login = login.replaceAll(" +", " ");
        return login;
    }

    private String getId(Map<String, Object> map) {
        return (String) map.get("id");
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOauthName() {
        return "facebook";
    }

    @Override
    protected Optional<UserAccount> findByOauthId(String oauthId) {
        return userAccountRepository.findByOauthIdentifiersFacebookId(oauthId);
    }

    @Override
    protected void setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        principal.getOauthIdentifiers().setFacebookId(oauthId);
    }

    @Override
    protected void setOauthIdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount.getOauthIdentifiers().setFacebookId(oauthId);
        userAccount = userAccountRepository.save(userAccount);
    }

    @Override
    protected UserAccount insertEntity(String oauthId, String login, Map<String, Object> map) {
        String maybeImageUrl = getAvatarUrl(map);
        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForFacebookInsert(oauthId, login, maybeImageUrl);
        userAccount = userAccountRepository.save(userAccount);
        LOGGER.info("Created facebook user id={} login='{}'", oauthId, login);

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
