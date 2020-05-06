package com.github.nkonev.blog.security;


import com.github.nkonev.blog.converter.UserAccountConverter;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.security.checks.BlogPostAuthenticationChecks;
import com.github.nkonev.blog.security.checks.BlogPreAuthenticationChecks;
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

import java.util.List;
import java.util.Map;
import java.util.Optional;

@Component
@Transactional
@Scope(proxyMode = ScopedProxyMode.TARGET_CLASS)
public class VkontakteOAuth2UserService extends AbstractOAuth2UserService implements OAuth2UserService<OAuth2UserRequest, OAuth2User> {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private BlogPreAuthenticationChecks blogPreAuthenticationChecks;

    @Autowired
    private BlogPostAuthenticationChecks blogPostAuthenticationChecks;


    private static final Logger LOGGER = LoggerFactory.getLogger(VkontakteOAuth2UserService.class);

    public static final String LOGIN_PREFIX = "vkontakte_";

    @Override
    public OAuth2User loadUser(OAuth2UserRequest userRequest) throws OAuth2AuthenticationException {
        OAuth2User oAuth2User = delegate.loadUser(userRequest);

        var map = oAuth2User.getAttributes();
        List l = (List) map.get("response");
        Map<String, Object> m = (Map<String, Object>) l.get(0);

        String vkontakteId = getId(m);
        Assert.notNull(vkontakteId, "vkontakteId cannot be null");

        UserAccountDetailsDTO resultPrincipal = mergeOauthIdToExistsUser(vkontakteId);
        if (resultPrincipal != null) {
            // ok
        } else {
            String login = getLogin(m);
            resultPrincipal = createOrGetExistsUser(vkontakteId, login, map);
        }

        blogPreAuthenticationChecks.check(resultPrincipal);
        blogPostAuthenticationChecks.check(resultPrincipal);
        return resultPrincipal;

    }

    private String getId(Map<String, Object> m) {
        return ((Integer) m.get("id")).toString();
    }

    private String getLogin(Map<String, Object> map) {
        String firstName = (String) map.get("first_name");
        String lastName = (String) map.get("last_name");
        String login = "";
        if (firstName!=null) {
            firstName = firstName.trim();
            login += firstName;
            login += " ";
        }
        if (lastName!=null) {
            lastName = lastName.trim();
            login += lastName;
        }
        Assert.hasLength(login, "vkontakte name cannot be null");
        login = login.trim();
        return login;
    }

    @Override
    protected Logger logger() {
        return LOGGER;
    }

    @Override
    protected String getOauthName() {
        return "vkontate";
    }

    @Override
    protected Optional<UserAccount> findByOauthId(String oauthId) {
        return userAccountRepository.findByOauthIdentifiersVkontakteId(oauthId);
    }

    @Override
    protected void setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId) {
        principal.getOauthIdentifiers().setVkontakteId(oauthId);
    }

    @Override
    protected void setOauthIdToEntity(Long id, String oauthId) {
        UserAccount userAccount = userAccountRepository.findById(id).orElseThrow();
        userAccount.getOauthIdentifiers().setVkontakteId(oauthId);
        userAccount = userAccountRepository.save(userAccount);
    }

    @Override
    protected UserAccount insertEntity(String oauthId, String login, Map<String, Object> oauthResourceServerResponse) {
        UserAccount userAccount = UserAccountConverter.buildUserAccountEntityForVkontakteInsert(oauthId, login);
        userAccount = userAccountRepository.save(userAccount);
        LOGGER.info("Created vkontakte user id={} login='{}'", oauthId, login);

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
