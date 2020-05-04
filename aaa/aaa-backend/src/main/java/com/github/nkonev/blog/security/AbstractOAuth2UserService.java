package com.github.nkonev.blog.security;

import com.github.nkonev.blog.converter.UserAccountConverter;
import com.github.nkonev.blog.dto.UserAccountDetailsDTO;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.exception.OAuth2IdConflictException;
import org.slf4j.Logger;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.client.userinfo.DefaultOAuth2UserService;

import java.util.Map;
import java.util.Optional;

public abstract class AbstractOAuth2UserService {

    final DefaultOAuth2UserService delegate = new DefaultOAuth2UserService();

    private boolean isAlreadyAuthenticated(){
        return SecurityContextHolder.getContext().getAuthentication()!=null && SecurityContextHolder.getContext().getAuthentication().getPrincipal() instanceof UserAccountDetailsDTO;
    }

    private UserAccountDetailsDTO getPrincipal() {
        return (UserAccountDetailsDTO) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
    }

    protected abstract Logger logger();

    protected abstract String getOauthName();

    protected abstract Optional<UserAccount> findByOauthId(String oauthId);

    protected abstract void setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId);

    protected abstract void setOauthIdToEntity(Long id, String oauthId);

    protected abstract UserAccount insertEntity(String oauthId, String login, Map<String, Object> oauthResourceServerResponse);

    protected abstract String getLoginPrefix();

    protected abstract Optional<UserAccount> findByUsername(String login);


    // @return notnull UserAccountDetailsDTO if was merged - so we should return it immediately from Extractor
    protected UserAccountDetailsDTO mergeOauthIdToExistsUser(String oauthId){
        if (isAlreadyAuthenticated()) {
            // we already authenticated - so it' s binding
            UserAccountDetailsDTO principal = getPrincipal();
            logger().info("Will merge {}Id to exists user '{}', id={}", getOauthName(), principal.getUsername(), principal.getId());

            Optional<UserAccount> maybeUserAccount = findByOauthId(oauthId);
            if (maybeUserAccount.isPresent() && !maybeUserAccount.get().getId().equals(principal.getId())){
                logger().error("With {}Id={} already present another user '{}', id={}", getOauthName(), oauthId, maybeUserAccount.get().getUsername(), maybeUserAccount.get().getId());
                throw new OAuth2IdConflictException("Somebody already taken this "+getOauthName()+" id="+oauthId+". " +
                        "If this is you and you want to merge your profiles please delete another profile and bind "+getOauthName()+" to this. If not please contact administrator.");
            }

            setOauthIdToPrincipal(principal, oauthId);

            setOauthIdToEntity(principal.getId(), oauthId);

            logger().info("{}Id successfully merged to exists user '{}', id={}", getOauthName(), principal.getUsername(), principal.getId());
            return principal;
        } else {
            return null;
        }
    }

    protected UserAccountDetailsDTO createOrGetExistsUser(String oauthId, String login, Map<String, Object> oauthResourceServerResponse) {
        UserAccount userAccount;
        Optional<UserAccount> userAccountOpt = findByOauthId(oauthId);
        if (!userAccountOpt.isPresent()){

            if (findByUsername(login).isPresent()){
                logger().info("User with login '{}' ({}) already present in database, so we' ll generate login", login, getOauthName());
                login = getLoginPrefix()+oauthId;
            }

            userAccount = insertEntity(oauthId, login, oauthResourceServerResponse);
        } else {
            userAccount = userAccountOpt.get();
        }

        return UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
    }

}
