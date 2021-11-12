package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.exception.OAuth2IdConflictException;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import org.slf4j.Logger;
import org.springframework.security.authentication.AbstractAuthenticationToken;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;

import java.util.Collection;
import java.util.Map;
import java.util.Optional;
import java.util.Set;

public abstract class AbstractOAuth2UserService {

    private boolean isAlreadyAuthenticated(){
        return SecurityContextHolder.getContext().getAuthentication()!=null && SecurityContextHolder.getContext().getAuthentication().getPrincipal() instanceof UserAccountDetailsDTO;
    }

    private UserAccountDetailsDTO getPrincipal() {
        return (UserAccountDetailsDTO) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
    }

    protected abstract Logger logger();

    protected abstract String getOauthName();

    protected abstract Optional<UserAccount> findByOauthId(String oauthId);

    protected abstract UserAccountDetailsDTO setOauthIdToPrincipal(UserAccountDetailsDTO principal, String oauthId);

    protected abstract void setOauthIdToEntity(Long id, String oauthId);

    protected abstract UserAccount insertEntity(String oauthId, String login, Map<String, Object> oauthResourceServerResponse, Set<String> roles);

    protected abstract String getLoginPrefix();

    protected abstract Optional<UserAccount> findByUsername(String login);


    // @return notnull UserAccountDetailsDTO if was merged - so we should return it immediately from Extractor
    protected UserAccountDetailsDTO mergeOauthIdToExistsUser(String oauthId){
        if (isAlreadyAuthenticated()) {
            // we already authenticated - so it' s binding
            UserAccountDetailsDTO principal = getPrincipal();
            logger().info("Will merge {}Id to exists user '{}', id={}", getOauthName(), principal.getUsername(), principal.getId());

            Optional<UserAccount> maybeUserAccount = findByOauthId(oauthId);
            if (maybeUserAccount.isPresent() && !maybeUserAccount.get().id().equals(principal.getId())){
                logger().error("With {}Id={} already present another user '{}', id={}", getOauthName(), oauthId, maybeUserAccount.get().username(), maybeUserAccount.get().id());
                throw new OAuth2IdConflictException("Somebody already taken this "+getOauthName()+" id="+oauthId+". " +
                        "If this is you and you want to merge your profiles please delete another profile and bind "+getOauthName()+" to this. If not please contact administrator.");
            }

            principal = setOauthIdToPrincipal(principal, oauthId);
            SecurityContextHolder.getContext().setAuthentication(new AaaAuthenticationToken(principal));

            setOauthIdToEntity(principal.getId(), oauthId);

            logger().info("{}Id successfully merged to exists user '{}', id={}", getOauthName(), principal.getUsername(), principal.getId());
            return principal;
        } else {
            return null;
        }
    }

    protected UserAccountDetailsDTO createOrGetExistsUser(String oauthId, String login, Map<String, Object> attributes, Set<String> roles) {
        UserAccount userAccount;
        Optional<UserAccount> userAccountOpt = findByOauthId(oauthId);
        if (!userAccountOpt.isPresent()){

            if (findByUsername(login).isPresent()){
                logger().info("User with login '{}' ({}) already present in database, so we' ll generate login", login, getOauthName());
                login = getLoginPrefix()+oauthId;
            }

            userAccount = insertEntity(oauthId, login, attributes, roles);
        } else {
            userAccount = userAccountOpt.get();
        }

        return UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
    }

}

