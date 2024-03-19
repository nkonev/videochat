package com.github.nkonev.aaa.services;

import com.github.nkonev.aaa.config.CustomConfig;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.*;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.exception.DataNotFoundException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.repository.redis.ChangeEmailConfirmationTokenRepository;
import com.github.nkonev.aaa.repository.spring.jdbc.UserListViewRepository;
import com.github.nkonev.aaa.security.*;
import com.github.nkonev.aaa.utils.PageUtils;
import jakarta.servlet.http.HttpSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.http.HttpHeaders;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.session.Session;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;

import java.time.Duration;
import java.util.*;
import java.util.function.Function;

import static com.github.nkonev.aaa.Constants.Headers.*;
import static com.github.nkonev.aaa.Constants.MAX_USERS_RESPONSE_LENGTH;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertRolesToStringList;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertToUserAccountDTO;

@Service
public class UserProfileService {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Autowired
    private CheckService userService;

    @Autowired
    private EventService notifier;

    @Autowired
    private UserRoleService userRoleService;

    @Autowired
    private UserListViewRepository userListViewRepository;

    @Autowired
    private ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    @Value("${custom.confirmation.change-email.token.ttl}")
    private Duration changeEmailConfirmationTokenTtl;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @Autowired
    private CustomConfig customConfig;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserProfileService.class);

    private Long getExpiresAt(HttpSession session) {
        Long expiresAt = null;
        if (session!=null) {
            expiresAt = session.getLastAccessedTime() + session.getMaxInactiveInterval()*1000;
        }
        return expiresAt;
    }

    /**
     *
     * @param userAccount
     * @return current logged in profile
     */
    @Transactional
    public UserSelfProfileDTO checkAuthenticated(UserAccountDetailsDTO userAccount, HttpSession session) {
        return UserAccountConverter.getUserSelfProfile(userAccount, userAccount.getLastLoginDateTime(), getExpiresAt(session));
    }

    @Transactional
    public HttpHeaders checkAuthenticatedInternal(UserAccountDetailsDTO userAccount, HttpSession session) {
        Long expiresAt = getExpiresAt(session);
        var dto = checkAuthenticated(userAccount, session);
        HttpHeaders headers = new HttpHeaders();
        headers.set(X_AUTH_USERNAME, Base64.getEncoder().encodeToString(dto.login().getBytes()));
        headers.set(X_AUTH_USER_ID, ""+userAccount.getId());
        headers.set(X_AUTH_EXPIRESIN, ""+expiresAt);
        headers.set(X_AUTH_SESSION_ID, session.getId());
        headers.set(X_AUTH_AVATAR, userAccount.getAvatar());
        convertRolesToStringList(userAccount.getRoles()).forEach(s -> {
            headers.add(X_AUTH_ROLE, s);
        });
        return headers;
    }


    @Transactional
    public List<UserAccountDTOExtended> searchUsers(
            UserAccountDetailsDTO userAccount,
            SearchUsersRequestDTO request
    ) {
        var searchString = request.searchString() != null ? request.searchString().trim() : "";
        var size = PageUtils.fixSize(request.size());

        var result = userListViewRepository.getUsers(size, request.startingFromItemId(), request.reverse(), request.hasHash(), searchString);

        return result.stream().map(getConvertToUserAccountDTO(userAccount)).toList();
    }

    @Transactional
    public Map<String, Boolean> filter(FilterUserRequest filterUserRequest) {
        var searchStringWithPercents = "%" + filterUserRequest.searchString() + "%";
        var found = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdIn(1, 0, searchStringWithPercents, List.of(filterUserRequest.userId()));
        return Map.of("found", !found.isEmpty());
    }

    @Transactional
    public SearchUsersResponseInternalDTO searchUsersInternal(SearchUsersRequestInternalDTO request) {
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(request.page()), PageUtils.fixSize(request.size()), Sort.Direction.ASC, "id");
        var searchString = request.searchString() != null ? request.searchString().trim() : "";

        final String forDbSearch = "%" + searchString + "%";
        List<UserAccount> resultPage;
        long count = 0;
        if (request.userIds() == null || request.userIds().isEmpty()) {
            resultPage = userAccountRepository.findByUsernameContainsIgnoreCase(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch);
            count = userAccountRepository.findByUsernameContainsIgnoreCaseCount(forDbSearch);
        } else {
            if (request.including()) {
                resultPage = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds());
                count = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdInCount(forDbSearch, request.userIds());
            } else {
                resultPage = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdNotIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds());
                count = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdNotInCount(forDbSearch, request.userIds());
            }
        }

        return new SearchUsersResponseInternalDTO(
            resultPage.stream().map(UserAccountConverter::convertToUserAccountDTO).toList(),
            count
        );
    }

    @Transactional
    public void requestUserOnline(List<Long> userIds) {
        List<UserOnlineResponse> usersOnline = aaaUserDetailsService.getUsersOnline(userIds);
        List<UserOnlineResponse> userOnlineResponses = usersOnline.stream().map(uo -> new UserOnlineResponse(uo.userId(), uo.online())).toList();
        notifier.notifyOnlineChanged(userOnlineResponses);
    }

    private Function<UserAccount, UserAccountDTOExtended> getConvertToUserAccountDTO(UserAccountDetailsDTO currentUser) {
        return userAccount -> userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(currentUser, userRoleService), userAccount);
    }

    @Transactional
    public Record getUser(
            Long userId,
            UserAccountDetailsDTO userAccountPrincipal
    ) {
        final UserAccount userAccountEntity = userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundException("User with id " + userId + " not found"));
        if (userAccountPrincipal != null && userAccountEntity.id().equals(userAccountPrincipal.getId())) {
            return UserAccountConverter.getUserSelfProfile(userAccountPrincipal, userAccountEntity.lastLoginDateTime(), null);
        } else {
            return convertToUserAccountDTO(userAccountEntity);
        }
    }

    @Transactional
    public List<UserAccountDTO> getUsersInternal(
        List<Long> userIds
    ) {
        if (userIds == null) {
            throw new BadRequestException("Cannot be null");
        }
        if (userIds.size() > MAX_USERS_RESPONSE_LENGTH) {
            throw new BadRequestException("Cannot be greater than " + MAX_USERS_RESPONSE_LENGTH);
        }
        var result = userAccountRepository.findByIdInOrderById(userIds).stream().map(UserAccountConverter::convertToUserAccountDTO).toList();
        return result;
    }

    @Transactional
    public UserSelfProfileDTO editNonEmpty(
            UserAccountDetailsDTO userAccount,
            EditUserDTO userAccountDTO,
            Language language,
            HttpSession httpSession
    ) {
        if (userAccount == null) {
            throw new RuntimeException("Not authenticated user can't edit any user account. It can occurs due inpatient refactoring.");
        }

        UserAccount exists = userAccountRepository.findById(userAccount.getId()).orElseThrow(() -> new RuntimeException("Authenticated user account not found in database"));

        // check email already present
        if (!userService.checkEmailIsFree(userAccountDTO, exists)) {
            // we care for email leak...
            return UserAccountConverter.getUserSelfProfile(UserAccountConverter.convertToUserAccountDetailsDTO(exists), userAccount.getLastLoginDateTime(), getExpiresAt(httpSession));
        }

        // check login already present
        userService.checkLoginIsFree(userAccountDTO, exists);

        var resp = UserAccountConverter.updateUserAccountEntityNotEmpty(userAccountDTO, exists, passwordEncoder);
        exists = resp.userAccount();
        if (resp.wasEmailSet()) {
            var changeEmailConfirmationToken = createChangeEmailConfirmationToken(exists.id());
            asyncEmailService.sendChangeEmailConfirmationToken(exists.newEmail(), changeEmailConfirmationToken, exists.username(), language);
        }
        exists = userAccountRepository.save(exists);

        SecurityUtils.convertAndSetToContext(httpSession, exists);

        notifier.notifyProfileUpdated(exists);

        return UserAccountConverter.getUserSelfProfile(UserAccountConverter.convertToUserAccountDetailsDTO(exists), userAccount.getLastLoginDateTime(), getExpiresAt(httpSession));
    }

    private ChangeEmailConfirmationToken createChangeEmailConfirmationToken(long userId) {
        var uuid = UUID.randomUUID();
        ChangeEmailConfirmationToken changeEmailConfirmationToken = new ChangeEmailConfirmationToken(uuid, userId, changeEmailConfirmationTokenTtl.getSeconds());
        return changeEmailConfirmationTokenRepository.save(changeEmailConfirmationToken);
    }

    @Transactional
    public String changeEmailConfirm(UUID uuid, HttpSession httpSession) {
        Optional<ChangeEmailConfirmationToken> userConfirmationTokenOptional = changeEmailConfirmationTokenRepository.findById(uuid);
        if (!userConfirmationTokenOptional.isPresent()) {
            return customConfig.getConfirmChangeEmailExitTokenNotFoundUrl();
        }
        ChangeEmailConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        UserAccount userAccount = userAccountRepository.findById(userConfirmationToken.userId()).orElseThrow();
        if (!StringUtils.hasLength(userAccount.newEmail())) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return customConfig.getConfirmChangeEmailExitSuccessUrl();
        }

        // check email already present
        if (!userService.checkEmailIsFree(userAccount.newEmail())) {
            LOGGER.info("Somebody has already taken this email {}", userAccount.newEmail());
            return customConfig.getConfirmChangeEmailExitSuccessUrl();
        }

        userAccount = userAccount.withEmail(userAccount.newEmail());
        userAccount = userAccount.withNewEmail(null);
        userAccount = userAccountRepository.save(userAccount);

        changeEmailConfirmationTokenRepository.deleteById(uuid);

        var auth = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityUtils.setToContext(httpSession, auth);

        notifier.notifyProfileUpdated(userAccount);

        return customConfig.getConfirmChangeEmailExitSuccessUrl();
    }

    @Transactional
    public void resendConfirmationChangeEmailToken(UserAccountDetailsDTO userAccount, Language language) {
        UserAccount theUserAccount = userAccountRepository.findById(userAccount.getId()).orElseThrow();
        if (!StringUtils.hasLength(theUserAccount.newEmail())) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return;
        }

        var changeEmailConfirmationToken = createChangeEmailConfirmationToken(theUserAccount.id());
        asyncEmailService.sendChangeEmailConfirmationToken(theUserAccount.newEmail(), changeEmailConfirmationToken, theUserAccount.username(), language);
    }

    @Transactional
    public Map<String, Session> mySessions(UserAccountDetailsDTO userDetails){
        return aaaUserDetailsService.getMySessions(userDetails);
    }

    @Transactional
    public List<UserOnlineResponse> getOnlineForUsers(List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }

    @Transactional
    public List<UserOnlineResponse> getOnlineForUsersInternal(List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }

    @Transactional
    public Map<String, Session> sessions(UserAccountDetailsDTO userAccount, long userId){
        return aaaUserDetailsService.getSessions(userId);
    }

    @Transactional
    public void killSessions(UserAccountDetailsDTO userAccount, long userId){
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.force_logged_out);
    }

    @Transactional
    public UserAccountDTOExtended setLocked(UserAccountDetailsDTO userAccountDetailsDTO, LockDTO lockDTO){
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(lockDTO.userId());
        if (lockDTO.lock()){
            aaaUserDetailsService.killSessions(lockDTO.userId(), ForceKillSessionsReasonType.user_locked);
        }
        userAccount = userAccount.withLocked(lockDTO.lock());
        userAccount = userAccountRepository.save(userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @Transactional
    public UserAccountDTOExtended setConfirmed(UserAccountDetailsDTO userAccountDetailsDTO, ConfirmDTO confirmDTO){
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(confirmDTO.userId());
        if (!confirmDTO.confirm()){
            aaaUserDetailsService.killSessions(confirmDTO.userId(), ForceKillSessionsReasonType.user_unconfirmed);
        }
        userAccount = userAccount.withConfirmed(confirmDTO.confirm());
        userAccount = userAccountRepository.save(userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @Transactional
    public void deleteUser(UserAccountDetailsDTO userAccountDetailsDTO, long userId){
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.user_deleted);
        notifier.notifyProfileDeleted(userId);
        userAccountRepository.deleteById(userId);
    }

    @Transactional
    public UserAccountDTOExtended setRole(UserAccountDetailsDTO userAccountDetailsDTO, long userId, UserRole role){
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount = userAccount.withRole(role);
        userAccount = userAccountRepository.save(userAccount);
        notifier.notifyProfileUpdated(userAccount);
        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @Transactional
    public void selfDeleteUser(UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.user_deleted);
        notifier.notifyProfileDeleted(userId);
        userAccountRepository.deleteById(userId);
    }

    @Transactional
    public UserSelfProfileDTO selfDeleteBindingOauth2Provider(
        UserAccountDetailsDTO userAccountDetailsDTO,
        String provider,
        HttpSession httpSession
    ){
        long userId = userAccountDetailsDTO.getId();
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        UserAccount.OAuth2Identifiers oAuth2Identifiers = switch (provider) {
            case OAuth2Providers.FACEBOOK -> userAccount.oauth2Identifiers().withFacebookId(null);
            case OAuth2Providers.VKONTAKTE -> userAccount.oauth2Identifiers().withVkontakteId(null);
            case OAuth2Providers.GOOGLE -> userAccount.oauth2Identifiers().withGoogleId(null);
            case OAuth2Providers.KEYCLOAK -> userAccount.oauth2Identifiers().withKeycloakId(null);
            default -> throw new RuntimeException("Wrong OAuth2 provider: " + provider);
        };
        userAccount = userAccount.withOauthIdentifiers(oAuth2Identifiers);
        userAccount = userAccountRepository.save(userAccount);
        SecurityUtils.convertAndSetToContext(httpSession, userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return UserAccountConverter.getUserSelfProfile(UserAccountConverter.convertToUserAccountDetailsDTO(userAccount), userAccountDetailsDTO.getLastLoginDateTime(), getExpiresAt(httpSession));
    }

    @Transactional
    public List<UserExists> getUsersExistInternal(
        List<Long> requestedUserIds
    ) {
        if (requestedUserIds == null) {
            throw new BadRequestException("Cannot be null");
        }
        if (requestedUserIds.isEmpty()) {
            return new ArrayList<>();
        }
        var existingUserIds = userAccountRepository.findUserIds(requestedUserIds);
        var result = new ArrayList<UserExists>();
        for (var userId : requestedUserIds) {
            var exists = existingUserIds.contains(userId);
            var ue = new UserExists(userId, exists);
            result.add(ue);
        }
        return result;
    }
}
