package com.github.nkonev.aaa.security;

import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.exception.OAuth2IdConflictException;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.security.checks.AaaPostAuthenticationChecks;
import com.github.nkonev.aaa.security.checks.AaaPreAuthenticationChecks;
import com.github.nkonev.aaa.services.EventService;
import org.slf4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.client.userinfo.OAuth2UserRequest;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.Assert;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import java.util.Map;
import java.util.Optional;
import java.util.Set;

import static com.github.nkonev.aaa.converter.UserAccountConverter.validateLengthAndTrimLogin;

public abstract class AbstractOAuth2UserService {

    @Autowired
    private AaaPreAuthenticationChecks aaaPreAuthenticationChecks;

    @Autowired
    private AaaPostAuthenticationChecks aaaPostAuthenticationChecks;

    @Autowired
    private EventService eventService;

    @Autowired
    private TransactionTemplate transactionTemplate;

    private boolean isAlreadyAuthenticated(){
        return SecurityContextHolder.getContext().getAuthentication()!=null && SecurityContextHolder.getContext().getAuthentication().getPrincipal() instanceof UserAccountDetailsDTO;
    }

    private UserAccountDetailsDTO getPrincipal() {
        return (UserAccountDetailsDTO) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
    }

    protected abstract Logger logger();

    protected abstract String getOAuth2Name();

    protected abstract Optional<UserAccount> findByOAuth2Id(String oauthId);

    protected abstract UserAccountDetailsDTO setOAuth2IdToPrincipal(UserAccountDetailsDTO principal, String oauthId);

    protected abstract UserAccount setOAuth2IdToEntity(Long id, String oauthId);

    protected abstract UserAccount insertEntity(String oauthId, String login, Map<String, Object> oauthResourceServerResponse, Set<String> roles);

    protected abstract String getLoginPrefix();

    protected abstract Optional<UserAccount> findByUsername(String login);

    private record MergeOAuthIdResponse(
        UserAccountDetailsDTO userAccountDetails,
        UserAccount userAccount
    ) { }

    // @return notnull UserAccountDetailsDTO if was merged - so we should return it immediately from Extractor
    private MergeOAuthIdResponse mergeOAuth2IdToExistsUser(String oauthId){
        if (isAlreadyAuthenticated()) {
            // we already authenticated - so it' s binding
            UserAccountDetailsDTO principal = getPrincipal();
            logger().info("Will merge {}Id to exists user '{}', id={}", getOAuth2Name(), principal.getUsername(), principal.getId());

            Optional<UserAccount> maybeUserAccount = findByOAuth2Id(oauthId);
            if (maybeUserAccount.isPresent() && !maybeUserAccount.get().id().equals(principal.getId())){
                logger().error("With {}Id={} already present another user '{}', id={}", getOAuth2Name(), oauthId, maybeUserAccount.get().username(), maybeUserAccount.get().id());
                throw new OAuth2IdConflictException("Somebody already taken this "+ getOAuth2Name()+" id="+oauthId+". " +
                        "If this is you and you want to merge your profiles please delete another profile and bind "+ getOAuth2Name()+" to this. If not please contact administrator.");
            }

            principal = setOAuth2IdToPrincipal(principal, oauthId);

            var userAccount = setOAuth2IdToEntity(principal.getId(), oauthId);

            logger().info("{}Id successfully merged to exists user '{}', id={}", getOAuth2Name(), principal.getUsername(), principal.getId());

            boolean setToSession = false;
            ServletRequestAttributes attributes = (ServletRequestAttributes) RequestContextHolder.getRequestAttributes();
            if (attributes != null) {
                var session = attributes.getRequest().getSession(false);
                if (session != null) {
                    SecurityUtils.setToContext(session, principal);
                    setToSession = true;
                }
            }
            if (!setToSession) {
                logger().warn("Unable to set changed principal to session");
            }

            return new MergeOAuthIdResponse(principal, userAccount);
        } else {
            return null;
        }
    }

    private record CreateOrGetExistsUserResponse(
        UserAccountDetailsDTO userAccountDetails,
        UserAccount userAccount,
        boolean created
    ) {}

    private CreateOrGetExistsUserResponse createOrGetExistsUser(String oauthId, String login, Map<String, Object> attributes, Set<String> roles) {
        UserAccount userAccount;
        login = validateLengthAndTrimLogin(login, true);
        Optional<UserAccount> userAccountOpt = findByOAuth2Id(oauthId);
        var created = false;
        if (!userAccountOpt.isPresent()){

            if (findByUsername(login).isPresent()){
                logger().info("User with login '{}' ({}) already present in database, so we' ll generate login", login, getOAuth2Name());
                login = getLoginPrefix()+oauthId;
            }

            userAccount = insertEntity(oauthId, login, attributes, roles);
            created = true;
        } else {
            userAccount = userAccountOpt.get();
        }

        return new CreateOrGetExistsUserResponse(UserAccountConverter.convertToUserAccountDetailsDTO(userAccount), userAccount, created);
    }

    abstract protected String getLogin(Map<String, Object> map);

    abstract protected String getId(Map<String, Object> map);

    private record ProcessIntermediateDTO (
        UserAccountDetailsDTO principal,
        UserAccount userAccount,
        boolean created
    ) {}

    protected UserAccountDetailsDTO process(Map<String, Object> map, OAuth2UserRequest userRequest) {
        String oauth2userId = getId(map);
        Assert.hasLength(oauth2userId, getOAuth2Name() + " id cannot be empty");

        ProcessIntermediateDTO processIntermediateDTO = transactionTemplate.execute(status -> {
            UserAccountDetailsDTO resultPrincipal;
            UserAccount userAccount;

            var created = false;
            var mergeOAuthToUserResponse = mergeOAuth2IdToExistsUser(oauth2userId);
            if (mergeOAuthToUserResponse != null) {
                // ok
                resultPrincipal = mergeOAuthToUserResponse.userAccountDetails();
                userAccount = mergeOAuthToUserResponse.userAccount();
            } else {
                String login = getLogin(map);
                var createOrGetResponse = createOrGetExistsUser(oauth2userId, login, map, getRoles(userRequest));
                created = createOrGetResponse.created();
                resultPrincipal = createOrGetResponse.userAccountDetails();
                userAccount = createOrGetResponse.userAccount();
            }

            aaaPreAuthenticationChecks.check(resultPrincipal);
            aaaPostAuthenticationChecks.check(resultPrincipal);
            return new ProcessIntermediateDTO(resultPrincipal, userAccount, created);
        });

        if (processIntermediateDTO.created()) {
            eventService.notifyProfileCreated(processIntermediateDTO.userAccount());
        }

        return processIntermediateDTO.principal();
    }

    protected Set<String> getRoles(OAuth2UserRequest userRequest) {
        return null;
    }
}

