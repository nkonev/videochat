package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
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
import com.github.nkonev.aaa.services.AsyncEmailService;
import com.github.nkonev.aaa.services.EventService;
import com.github.nkonev.aaa.services.OAuth2ProvidersService;
import com.github.nkonev.aaa.services.UserService;
import com.github.nkonev.aaa.utils.PageUtils;
import jakarta.servlet.http.HttpSession;
import jakarta.validation.Valid;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.session.Session;
import org.springframework.stereotype.Controller;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.StringUtils;
import org.springframework.web.bind.annotation.*;

import java.time.Duration;
import java.util.*;
import java.util.function.Function;

import static com.github.nkonev.aaa.Constants.Headers.*;
import static com.github.nkonev.aaa.Constants.MAX_USERS_RESPONSE_LENGTH;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertRolesToStringList;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertToUserAccountDTO;

/**
 * Created by nik on 08.06.17.
 */
@Controller
@Transactional
public class UserProfileController {

    @Autowired
    private UserAccountRepository userAccountRepository;

    @Autowired
    private PasswordEncoder passwordEncoder;

    @Autowired
    private AaaUserDetailsService aaaUserDetailsService;

    @Autowired
    private UserAccountConverter userAccountConverter;

    @Autowired
    private UserService userService;

    @Autowired
    private EventService notifier;

    @Autowired
    private OAuth2ProvidersService oAuth2ProvidersService;

    @Autowired
    private UserRoleService userRoleService;

    @Autowired
    private UserListViewRepository userListViewRepository;

    @Autowired
    private ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    @Value("${custom.change-email.token.ttl}")
    private Duration changeEmailConfirmationTokenTtl;

    @Autowired
    private AsyncEmailService asyncEmailService;

    @Autowired
    private CustomConfig customConfig;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserProfileController.class);

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
    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE, produces = MediaType.APPLICATION_JSON_VALUE)
    public com.github.nkonev.aaa.dto.UserSelfProfileDTO checkAuthenticated(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting external user profile");
        Long expiresAt = getExpiresAt(session);
        return UserAccountConverter.getUserSelfProfile(userAccount, userAccount.getLastLoginDateTime(), expiresAt);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = {Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE, Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH}, produces = MediaType.APPLICATION_JSON_VALUE)
    public HttpHeaders checkAuthenticatedInternal(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting internal user profile");
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


    record SearchUsersRequestInternalDto(
        int page,
        int size,
        List<Long> userIds,
        String searchString,
        boolean including
    ) {}


    record SearchUsersResponseInternalDto(
        List<com.github.nkonev.aaa.dto.UserAccountDTO> users,
        long count
    ) {}

    record SearchUsersRequestDto(
        int size,
        long startingFromItemId,
        boolean reverse,
        boolean hasHash,
        String searchString
    ) {}


    @ResponseBody
    @CrossOrigin(origins="*", methods = RequestMethod.POST)
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.SEARCH)
    public List<com.github.nkonev.aaa.dto.UserAccountDTOExtended> searchUsers(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody SearchUsersRequestDto request
    ) {
        LOGGER.info("Searching users external");
        var searchString = request.searchString != null ? request.searchString.trim() : "";
        var size = PageUtils.fixSize(request.size);

        var result = userListViewRepository.getUsers(size, request.startingFromItemId, request.reverse, request.hasHash, searchString);

        return result.stream().map(getConvertToUserAccountDTO(userAccount)).toList();
    }

    @ResponseBody
    @CrossOrigin(origins="*", methods = RequestMethod.POST)
    @PostMapping(Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.SEARCH)
    public SearchUsersResponseInternalDto searchUsersInternal(@RequestBody SearchUsersRequestInternalDto request) {
        LOGGER.info("Searching users internal");
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(request.page), PageUtils.fixSize(request.size), Sort.Direction.ASC, "id");
        var searchString = request.searchString != null ? request.searchString.trim() : "";

        final String forDbSearch = "%" + searchString + "%";
        List<UserAccount> resultPage;
        long count = 0;
        if (request.userIds == null || request.userIds.isEmpty()) {
            resultPage = userAccountRepository.findByUsernameContainsIgnoreCase(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch);
            count = userAccountRepository.findByUsernameContainsIgnoreCaseCount(forDbSearch);
        } else {
            if (request.including) {
                resultPage = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds);
                count = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdInCount(forDbSearch, request.userIds);
            } else {
                resultPage = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdNotIn(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch, request.userIds);
                count = userAccountRepository.findByUsernameContainsIgnoreCaseAndIdNotInCount(forDbSearch, request.userIds);
            }
        }

        return new SearchUsersResponseInternalDto(
            resultPage.stream().map(UserAccountConverter::convertToUserAccountDTO).toList(),
            count
        );
    }

    @ResponseBody
    @PutMapping(Constants.Urls.INTERNAL_API+Constants.Urls.USER + Constants.Urls.REQUEST_FOR_ONLINE)
    public void requestUserOnline(@RequestParam(value = "userId") List<Long> userIds) {
        List<UserOnlineResponse> usersOnline = aaaUserDetailsService.getUsersOnline(userIds);
        List<UserOnlineResponse> userOnlineResponses = usersOnline.stream().map(uo -> new UserOnlineResponse(uo.userId, uo.online)).toList();
        notifier.notifyOnlineChanged(userOnlineResponses);
    }

    private Function<UserAccount, com.github.nkonev.aaa.dto.UserAccountDTOExtended> getConvertToUserAccountDTO(UserAccountDetailsDTO currentUser) {
        return userAccount -> userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(currentUser, userRoleService), userAccount);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.USER_ID)
    public Record getUser(
            @PathVariable(value = Constants.PathVariables.USER_ID) Long userId,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        final UserAccount userAccountEntity = userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundException("User with id " + userId + " not found"));
        if (userAccountPrincipal != null && userAccountEntity.id().equals(userAccountPrincipal.getId())) {
            return UserAccountConverter.getUserSelfProfile(userAccountPrincipal, userAccountEntity.lastLoginDateTime(), null);
        } else {
            return convertToUserAccountDTO(userAccountEntity);
        }
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<UserAccountDTO> getUsersInternal(
        @RequestParam(value = "userId") List<Long> userIds
    ) {
        LOGGER.info("Requesting internal users {}", userIds);
        if (userIds == null) {
            throw new BadRequestException("Cannot be null");
        }
        if (userIds.size() > MAX_USERS_RESPONSE_LENGTH) {
            throw new BadRequestException("Cannot be greater than " + MAX_USERS_RESPONSE_LENGTH);
        }
        var result = userAccountRepository.findByIdInOrderById(userIds).stream().map(UserAccountConverter::convertToUserAccountDTO).toList();
        return result;
    }

    @ResponseBody
    @PatchMapping(Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE)
    @PreAuthorize("isAuthenticated()")
    public com.github.nkonev.aaa.dto.EditUserDTO editNonEmpty(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody @Valid com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO,
            @RequestParam(defaultValue = Language.DEFAULT) Language language,
            HttpSession httpSession
    ) {
        if (userAccount == null) {
            throw new RuntimeException("Not authenticated user can't edit any user account. It can occurs due inpatient refactoring.");
        }

        UserAccount exists = userAccountRepository.findById(userAccount.getId()).orElseThrow(() -> new RuntimeException("Authenticated user account not found in database"));

        // check email already present
        if (!userService.checkEmailIsFree(userAccountDTO, exists))
            return UserAccountConverter.convertToEditUserDto(exists); // we care for email leak...

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

        return UserAccountConverter.convertToEditUserDto(exists);
    }

    private ChangeEmailConfirmationToken createChangeEmailConfirmationToken(long userId) {
        var uuid = UUID.randomUUID();
        ChangeEmailConfirmationToken changeEmailConfirmationToken = new ChangeEmailConfirmationToken(uuid, userId, changeEmailConfirmationTokenTtl.getSeconds());
        return changeEmailConfirmationTokenRepository.save(changeEmailConfirmationToken);
    }

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.CHANGE_EMAIL_CONFIRM)
    public String confirm(@RequestParam(Constants.Urls.UUID) UUID uuid, HttpSession httpSession) {
        Optional<ChangeEmailConfirmationToken> userConfirmationTokenOptional = changeEmailConfirmationTokenRepository.findById(uuid);
        if (!userConfirmationTokenOptional.isPresent()) {
            return "redirect:" + customConfig.getConfirmChangeEmailExitTokenNotFoundUrl();
        }
        ChangeEmailConfirmationToken userConfirmationToken = userConfirmationTokenOptional.get();
        UserAccount userAccount = userAccountRepository.findById(userConfirmationToken.userId()).orElseThrow();
        if (!StringUtils.hasLength(userAccount.newEmail())) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return "redirect:" + customConfig.getConfirmChangeEmailExitSuccessUrl();
        }

        userAccount = userAccount.withEmail(userAccount.newEmail());
        userAccount = userAccount.withNewEmail(null);
        userAccount = userAccountRepository.save(userAccount);

        changeEmailConfirmationTokenRepository.deleteById(uuid);

        var auth = UserAccountConverter.convertToUserAccountDetailsDTO(userAccount);
        SecurityUtils.setToContext(httpSession, auth);

        notifier.notifyProfileUpdated(userAccount);

        return "redirect:" + customConfig.getConfirmChangeEmailExitSuccessUrl();
    }

    @PreAuthorize("isAuthenticated()")
    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CHANGE_EMAIL_CONFIRM)
    @ResponseBody
    public void resendConfirmationChangeEmailToken(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        UserAccount theUserAccount = userAccountRepository.findById(userAccount.getId()).orElseThrow();
        if (!StringUtils.hasLength(theUserAccount.newEmail())) {
            LOGGER.info("Somebody attempts confirm again changing the email of {}, but there is no new email", userAccount);
            return;
        }

        var changeEmailConfirmationToken = createChangeEmailConfirmationToken(theUserAccount.id());
        asyncEmailService.sendChangeEmailConfirmationToken(theUserAccount.newEmail(), changeEmailConfirmationToken, theUserAccount.username(), language);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS+"/my")
    public Map<String, Session> mySessions(@AuthenticationPrincipal UserAccountDetailsDTO userDetails){
        return aaaUserDetailsService.getMySessions(userDetails);
    }

    public record UserOnlineResponse (long userId, boolean online) {}

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsers(@RequestParam(value = "userId") List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.INTERNAL_API + Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsersInternal(@RequestParam(value = "userId") List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.hasSessionManagementPermission(#userAccount)")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS)
    public Map<String, Session> sessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        return aaaUserDetailsService.getSessions(userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.hasSessionManagementPermission(#userAccount)")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS)
    public void killSessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.force_logged_out);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canLock(#userAccountDetailsDTO, #lockDTO)")
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.LOCK)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setLocked(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody com.github.nkonev.aaa.dto.LockDTO lockDTO){
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(lockDTO.userId());
        if (lockDTO.lock()){
            aaaUserDetailsService.killSessions(lockDTO.userId(), ForceKillSessionsReasonType.user_locked);
        }
        userAccount = userAccount.withLocked(lockDTO.lock());
        userAccount = userAccountRepository.save(userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canConfirm(#userAccountDetailsDTO, #confirmDTO)")
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.CONFIRM)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setConfirmed(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody com.github.nkonev.aaa.dto.ConfirmDTO confirmDTO){
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(confirmDTO.userId());
        if (!confirmDTO.confirm()){
            aaaUserDetailsService.killSessions(confirmDTO.userId(), ForceKillSessionsReasonType.user_unconfirmed);
        }
        userAccount = userAccount.withConfirmed(confirmDTO.confirm());
        userAccount = userAccountRepository.save(userAccount);

        notifier.notifyProfileUpdated(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canDelete(#userAccountDetailsDTO, #userId)")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER)
    public void deleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam("userId") long userId){
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.user_deleted);
        notifier.notifyProfileDeleted(userId);
        userService.deleteUser(userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canChangeRole(#userAccountDetailsDTO, #userId)")
    @PutMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.ROLE)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setRole(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam long userId, @RequestParam UserRole role){
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount = userAccount.withRole(role);
        userAccount = userAccountRepository.save(userAccount);
        notifier.notifyProfileUpdated(userAccount);
        return userAccountConverter.convertToUserAccountDTOExtended(PrincipalToCheck.ofUserAccount(userAccountDetailsDTO, userRoleService), userAccount);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.ROLE)
    public List<UserRole> getAllRoles() {
        return Arrays.stream(UserRole.values()).toList();
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canSelfDelete(#userAccountDetailsDTO)")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE)
    public void selfDeleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        aaaUserDetailsService.killSessions(userId, ForceKillSessionsReasonType.user_deleted);
        notifier.notifyProfileDeleted(userId);
        userService.deleteUser(userId);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE+"/{provider}")
    public void selfDeleteBindingOauth2Provider(
        @AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO,
        @PathVariable("provider") String provider,
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
    }

    @ResponseBody
    @GetMapping(Constants.Urls.PUBLIC_API + "/oauth2/providers")
    public Set<String> availableOauth2Providers() {
        return oAuth2ProvidersService.availableOauth2Providers();
    }

    record UserExists (
        long userId,
        boolean exists
    ) {}


    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+"/exist")
    public List<UserExists> getUsersExistInternal(
        @RequestParam(value = "userId") List<Long> requestedUserIds
    ) {
        LOGGER.info("Requesting internal users exist {}", requestedUserIds);
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
