package name.nkonev.aaa.services;

import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.converter.UserAccountConverter;
import name.nkonev.aaa.dto.*;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.entity.redis.ChangeEmailConfirmationToken;
import name.nkonev.aaa.exception.BadRequestException;
import name.nkonev.aaa.exception.DataNotFoundException;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.repository.redis.ChangeEmailConfirmationTokenRepository;
import name.nkonev.aaa.repository.spring.jdbc.UserListViewRepository;
import name.nkonev.aaa.security.*;
import name.nkonev.aaa.utils.PageUtils;
import jakarta.servlet.http.HttpSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.http.HttpHeaders;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.session.Session;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

import java.util.*;
import java.util.function.Function;

import static name.nkonev.aaa.Constants.*;
import static name.nkonev.aaa.Constants.Headers.*;
import static name.nkonev.aaa.converter.UserAccountConverter.*;

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
    private CheckService checkService;

    @Autowired
    private EventService notifier;

    @Autowired
    private UserRoleService userRoleService;

    @Autowired
    private UserListViewRepository userListViewRepository;

    @Autowired
    private ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @Autowired
    private AaaProperties aaaProperties;

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
    public UserSelfProfileDTO getProfile(UserAccountDetailsDTO userAccount, HttpSession session) {
        return UserAccountConverter.getUserSelfProfile(userAccount, userAccount.getLastLoginDateTime(), getExpiresAt(session));
    }

    @Transactional
    public HttpHeaders checkAuthenticatedInternal(UserAccountDetailsDTO userAccount, HttpSession session) {
        Long expiresAt = getExpiresAt(session);
        var dto = getProfile(userAccount, session);
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
        var searchString = filterUserRequest.searchString() != null ? filterUserRequest.searchString().trim() : "";
        var searchStringWithPercents = "%" + searchString + "%";
        var found = userListViewRepository.findByUsernameContainsIgnoreCaseAndIdIn(1, 0, searchStringWithPercents, List.of(filterUserRequest.userId()));
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
            resultPage = userListViewRepository.findByUsernameContainsIgnoreCase(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch);
            count = userListViewRepository.findByUsernameContainsIgnoreCaseCount(forDbSearch);
        } else {
            if (request.including()) {
                resultPage = userListViewRepository.findByUsernameContainsIgnoreCaseAndIdIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds());
                count = userListViewRepository.findByUsernameContainsIgnoreCaseAndIdInCount(forDbSearch, request.userIds());
            } else {
                resultPage = userListViewRepository.findByUsernameContainsIgnoreCaseAndIdNotIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds());
                count = userListViewRepository.findByUsernameContainsIgnoreCaseAndIdNotInCount(forDbSearch, request.userIds());
            }
        }

        return new SearchUsersResponseInternalDTO(
            resultPage.stream().map(UserAccountConverter::convertToUserAccountDTO).toList(),
            count
        );
    }

    @Transactional
    public void requestUserOnline(List<Long> userIds) {
        List<Long> userIdsReal;
        if (userIds.size() > USERS_ONLINE_LENGTH) {
            userIdsReal = userIds.stream().limit(USERS_ONLINE_LENGTH).toList();
            LOGGER.info("Cutting {} userIds to {}", userIds.size(), USERS_ONLINE_LENGTH);
        } else {
            userIdsReal = userIds;
        }

        List<UserOnlineResponse> usersOnline = aaaUserDetailsService.getUsersOnline(userIdsReal);
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

        userAccountDTO = UserAccountConverter.normalize(userAccountDTO, false);

        UserAccount exists = userAccountRepository.findById(userAccount.getId()).orElseThrow(() -> new RuntimeException("Authenticated user account not found in database"));

        // check email already present
        if (userAccountDTO.email() != null && exists.email() != null && !exists.email().equals(userAccountDTO.email()) && !checkService.checkEmailIsFree(userAccountDTO.email())) {
            LOGGER.info("User {} tries to take an email {} which is already busy", userAccount.getId(), userAccountDTO.email());
            // we care for email leak...
            return UserAccountConverter.getUserSelfProfile(userAccountConverter.convertToUserAccountDetailsDTO(exists), userAccount.getLastLoginDateTime(), getExpiresAt(httpSession));
        }

        // check login already present
        if (userAccountDTO.login() != null && !exists.username().equals(userAccountDTO.login())) {
            checkService.checkLoginIsFree(userAccountDTO.login());
        }

        var resp = userAccountConverter.updateUserAccountEntityNotEmpty(userAccountDTO, exists, passwordEncoder);
        exists = resp.userAccount();
        exists = userAccountRepository.save(exists);

        switch (resp.action()) {
            case NEW_EMAIL_WAS_SET -> {
                var changeEmailConfirmationToken = createChangeEmailConfirmationToken(exists.id(), resp.newEmail());
                asyncEmailService.sendChangeEmailConfirmationToken(changeEmailConfirmationToken, exists.username(), language);
            }
            case SHOULD_REMOVE_NEW_EMAIL -> changeEmailConfirmationTokenRepository.deleteById(userAccount.getId());
        }

        SecurityUtils.convertAndSetToContext(userAccountConverter, httpSession, exists);

        notifier.notifyProfileUpdated(exists);

        return UserAccountConverter.getUserSelfProfile(userAccountConverter.convertToUserAccountDetailsDTO(exists), userAccount.getLastLoginDateTime(), getExpiresAt(httpSession));
    }

    private ChangeEmailConfirmationToken createChangeEmailConfirmationToken(long userId, String newEmail) {
        var uuid = UUID.randomUUID();
        ChangeEmailConfirmationToken changeEmailConfirmationToken = new ChangeEmailConfirmationToken(userId, uuid, newEmail, aaaProperties.confirmation().changeEmail().token().ttl().getSeconds());
        return changeEmailConfirmationTokenRepository.save(changeEmailConfirmationToken);
    }

    @Transactional
    public String changeEmailConfirm(long userId, UUID uuid, HttpSession httpSession) {
        Optional<ChangeEmailConfirmationToken> userConfirmationTokenOptional = changeEmailConfirmationTokenRepository.findById(userId);
        if (userConfirmationTokenOptional.isEmpty()) {
            LOGGER.info("For uuid {}, change email token is not found", uuid);
            return aaaProperties.confirmChangeEmailExitTokenNotFoundUrl();
        }
        ChangeEmailConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        if (!userConfirmationToken.uuid().equals(uuid)) {
            LOGGER.info("For uuid {}, change email token has the different uuid, exiting", uuid);
            return aaaProperties.confirmChangeEmailExitTokenNotFoundUrl();
        }

        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        var newEmail = userConfirmationToken.newEmail();
        if (!StringUtils.hasLength(newEmail)) {
            LOGGER.warn("Token has no email userId {}", userAccount.id());
            return aaaProperties.confirmChangeEmailExitSuccessUrl();
        }

        // check email already present
        if (!checkService.checkEmailIsFree(newEmail)) {
            LOGGER.info("Somebody has already taken this email {}", newEmail);
            return aaaProperties.confirmChangeEmailExitSuccessUrl();
        }

        userAccount = userAccount.withEmail(newEmail);
        userAccount = userAccountRepository.save(userAccount);

        changeEmailConfirmationTokenRepository.deleteById(userId);

        var auth = userAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityUtils.setToContext(httpSession, auth);

        notifier.notifyProfileUpdated(userAccount);

        return aaaProperties.confirmChangeEmailExitSuccessUrl();
    }

    @Transactional
    public void resendConfirmationChangeEmailToken(UserAccountDetailsDTO userAccount, Language language) {
        UserAccount theUserAccount = userAccountRepository.findById(userAccount.getId()).orElseThrow();
        Optional<ChangeEmailConfirmationToken> userConfirmationTokenOptional = changeEmailConfirmationTokenRepository.findById(userAccount.getId());

        if (userConfirmationTokenOptional.isEmpty()) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return;
        }

        var previousToken = userConfirmationTokenOptional.get();
        if (!StringUtils.hasLength(previousToken.newEmail())) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return;
        }

        var changeEmailConfirmationToken = createChangeEmailConfirmationToken(theUserAccount.id(), previousToken.newEmail());
        asyncEmailService.sendChangeEmailConfirmationToken(changeEmailConfirmationToken, theUserAccount.username(), language);
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
    public void killSessions(UserAccountDetailsDTO userAccount, long userId, HttpSession httpSession){
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.force_logged_out, httpSession.getId(), userAccount.getId());
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
    public UserAccountDTOExtended setRole(UserAccountDetailsDTO userAccountDetailsDTO, long userId, Set<UserRole> roles){
        Assert.isTrue(!roles.isEmpty(), "Role should be");
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount = userAccount.withRoles(roles.toArray(new UserRole[0]));
        userAccount = userAccountRepository.save(userAccount);
        if (!userAccountDetailsDTO.getId().equals(userId)) {
            aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.user_roles_changed);
        }
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
        SecurityUtils.convertAndSetToContext(userAccountConverter, httpSession, userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return UserAccountConverter.getUserSelfProfile(userAccountConverter.convertToUserAccountDetailsDTO(userAccount), userAccountDetailsDTO.getLastLoginDateTime(), getExpiresAt(httpSession));
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
