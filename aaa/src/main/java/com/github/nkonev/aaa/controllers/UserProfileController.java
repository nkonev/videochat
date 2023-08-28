package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.exception.DataNotFoundException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaUserDetailsService;
import com.github.nkonev.aaa.security.OAuth2Providers;
import com.github.nkonev.aaa.services.EventService;
import com.github.nkonev.aaa.services.UserService;
import com.github.nkonev.aaa.utils.PageUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.security.oauth2.client.OAuth2ClientProperties;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.session.Session;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.*;

import javax.servlet.http.HttpSession;
import javax.validation.Valid;
import java.util.*;
import java.util.function.Function;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.Constants.Headers.*;
import static com.github.nkonev.aaa.Constants.MAX_USERS_RESPONSE_LENGTH;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertRolesToStringList;
import static java.util.stream.Stream.ofNullable;

/**
 * Created by nik on 08.06.17.
 */
@RestController
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
    private ObjectProvider<OAuth2ClientProperties> oAuth2ClientProperties;

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
    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.API+Constants.Urls.PROFILE, produces = MediaType.APPLICATION_JSON_VALUE)
    public com.github.nkonev.aaa.dto.UserSelfProfileDTO checkAuthenticated(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting external user profile");
        Long expiresAt = getExpiresAt(session);
        return UserAccountConverter.getUserSelfProfile(userAccount, userAccount.getLastLoginDateTime(), expiresAt);
    }

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


    @GetMapping(value = Constants.Urls.API+Constants.Urls.USER)
    public com.github.nkonev.aaa.dto.Wrapper<com.github.nkonev.aaa.dto.UserAccountDTOExtended> getUsers(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestParam(value = "page", required=false, defaultValue = "0") int page,
            @RequestParam(value = "size", required=false, defaultValue = "0") int size
    ) {
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(page), PageUtils.fixSize(size), Sort.Direction.ASC, "id");

        List<UserAccount> resultPage = userAccountRepository.findAll(springDataPage).toList();
        long resultPageCount = userAccountRepository.count();

        return new com.github.nkonev.aaa.dto.Wrapper<>(
                resultPageCount,
                resultPage.stream().map(getConvertToUserAccountDTO(userAccount)).collect(Collectors.toList())
        );
    }

    record SearchUsersRequestDto(
        int page,
        int size,
        List<Long> userIds,
        String searchString,
        boolean including
    ) {}

    record SearchUsersResponseDto(
            List<com.github.nkonev.aaa.dto.UserAccountDTOExtended> users,
            long count
    ) {}

    @CrossOrigin(origins="*", methods = RequestMethod.POST)
    @PostMapping(value = {
            Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.SEARCH,
            Constants.Urls.API+Constants.Urls.USER+Constants.Urls.SEARCH
    })
    public SearchUsersResponseDto searchUserInternal(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody SearchUsersRequestDto request
    ) {
        LOGGER.info("Searching users");
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(request.page), PageUtils.fixSize(request.size), Sort.Direction.ASC, "id");
        var searchString = request.searchString.trim();

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

        return new SearchUsersResponseDto(
                resultPage.stream().map(getConvertToUserAccountDTO(userAccount)).collect(Collectors.toList()),
                count
        );
    }

    @PutMapping(Constants.Urls.INTERNAL_API+Constants.Urls.USER + Constants.Urls.REQUEST_FOR_ONLINE)
    public void requestUserOnline(@RequestParam(value = "userId") List<Long> userIds) {
        List<UserOnlineResponse> usersOnline = aaaUserDetailsService.getUsersOnline(userIds);
        List<UserOnlineResponse> userOnlineResponses = usersOnline.stream().map(uo -> new UserOnlineResponse(uo.userId, uo.online)).toList();
        notifier.notifyOnlineChanged(userOnlineResponses);
    }

    private Function<UserAccount, com.github.nkonev.aaa.dto.UserAccountDTOExtended> getConvertToUserAccountDTO(UserAccountDetailsDTO currentUser) {
        return userAccount -> userAccountConverter.convertToUserAccountDTOExtended(currentUser, userAccount);
    }

    @GetMapping(value = Constants.Urls.API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<Record> getUsers(
            @RequestParam(value = "userId") List<Long> userIds,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
        ) {
        if (userIds == null) {
            throw new BadRequestException("Cannot be null");
        }
        if (userIds.size() > MAX_USERS_RESPONSE_LENGTH) {
            throw new BadRequestException("Cannot be greater than " + MAX_USERS_RESPONSE_LENGTH);
        }
        List<Record> result = new ArrayList<>();
        for (UserAccount userAccountEntity: userAccountRepository.findByIdInOrderById(userIds)) {
            if (userAccountPrincipal != null && userAccountPrincipal.getId().equals(userAccountEntity.id())) {
                result.add(UserAccountConverter.getUserSelfProfile(userAccountPrincipal, userAccountEntity.lastLoginDateTime(), null));
            } else {
                result.add(userAccountConverter.convertToUserAccountDTO(userAccountEntity));
            }
        }
        return result;
    }

    @GetMapping(value = Constants.Urls.API+Constants.Urls.USER+Constants.Urls.USER_ID)
    public Record getUser(
            @PathVariable(value = Constants.PathVariables.USER_ID) Long userId,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        final UserAccount userAccountEntity = userAccountRepository.findById(userId).orElseThrow(() -> new DataNotFoundException("User with id " + userId + " not found"));
        if (userAccountPrincipal != null && userAccountPrincipal.getId().equals(userAccountEntity.id())) {
            return UserAccountConverter.getUserSelfProfile(userAccountPrincipal, userAccountEntity.lastLoginDateTime(), null);
        } else {
            return userAccountConverter.convertToUserAccountDTO(userAccountEntity);
        }
    }

    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<Record> getUserInternal(
            @RequestParam(value = "userId") List<Long> userIds,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        LOGGER.info("Requesting internal users {}", userIds);
        return getUsers(userIds, userAccountPrincipal);
    }

    @PostMapping(Constants.Urls.API+Constants.Urls.PROFILE)
    @PreAuthorize("isAuthenticated()")
    public com.github.nkonev.aaa.dto.EditUserDTO editProfile(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody @Valid com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO
    ) {
        if (userAccount == null) {
            throw new RuntimeException("Not authenticated user can't edit any user account. It can occurs due inpatient refactoring.");
        }

        UserAccount exists = findUserAccount(userAccount);

        userAccountDTO = UserAccountConverter.trimAndValidateNonAouth2Login(userAccountDTO);

        // check email already present
        if (!userService.checkEmailIsFree(userAccountDTO, exists)) return UserAccountConverter.convertToEditUserDto(exists);

        // check login already present
        userService.checkLoginIsFree(userAccountDTO, exists);

        exists = UserAccountConverter.updateUserAccountEntity(userAccountDTO, exists, passwordEncoder);
        exists = userAccountRepository.save(exists);

        aaaUserDetailsService.refreshUserDetails(exists);
        notifier.notifyProfileUpdated(exists);

        return UserAccountConverter.convertToEditUserDto(exists);
    }

    private UserAccount findUserAccount(@AuthenticationPrincipal UserAccountDetailsDTO userAccount) {
        return userAccountRepository.findById(userAccount.getId()).orElseThrow(() -> new RuntimeException("Authenticated user account not found in database"));
    }

    @PatchMapping(Constants.Urls.API+Constants.Urls.PROFILE)
    @PreAuthorize("isAuthenticated()")
    public com.github.nkonev.aaa.dto.EditUserDTO editNonEmpty(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody @Valid com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO
    ) {
        if (userAccount == null) {
            throw new RuntimeException("Not authenticated user can't edit any user account. It can occurs due inpatient refactoring.");
        }

        UserAccount exists = findUserAccount(userAccount);

        // check email already present
        if (!userService.checkEmailIsFree(userAccountDTO, exists))
            return UserAccountConverter.convertToEditUserDto(exists); // we care for email leak...

        // check login already present
        userService.checkLoginIsFree(userAccountDTO, exists);

        exists = UserAccountConverter.updateUserAccountEntityNotEmpty(userAccountDTO, exists, passwordEncoder);
        exists = userAccountRepository.save(exists);

        aaaUserDetailsService.refreshUserDetails(exists);

        notifier.notifyProfileUpdated(exists);

        return UserAccountConverter.convertToEditUserDto(exists);
    }

    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.API+Constants.Urls.SESSIONS+"/my")
    public Map<String, Session> mySessions(@AuthenticationPrincipal UserAccountDetailsDTO userDetails){
        return aaaUserDetailsService.getMySessions(userDetails);
    }

    public record UserOnlineResponse (long userId, boolean online) {}

    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.API+Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsers(@RequestParam(value = "userId") List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }
    @GetMapping(Constants.Urls.INTERNAL_API + Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsersInternal(@RequestParam(value = "userId") List<Long> userIds){
        return aaaUserDetailsService.getUsersOnline(userIds);
    }

    @PreAuthorize("@aaaSecurityService.hasSessionManagementPermission(#userAccount)")
    @GetMapping(Constants.Urls.API+Constants.Urls.SESSIONS)
    public Map<String, Session> sessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        return aaaUserDetailsService.getSessions(userId);
    }

    @PreAuthorize("@aaaSecurityService.hasSessionManagementPermission(#userAccount)")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.SESSIONS)
    public void killSessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        aaaUserDetailsService.killSessions(userId);
    }

    @PreAuthorize("@aaaSecurityService.canLock(#userAccountDetailsDTO, #lockDTO)")
    @PostMapping(Constants.Urls.API+Constants.Urls.USER + Constants.Urls.LOCK)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setLocked(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody com.github.nkonev.aaa.dto.LockDTO lockDTO){
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(lockDTO.userId());
        if (lockDTO.lock()){
            aaaUserDetailsService.killSessions(lockDTO.userId());
        }
        userAccount = userAccount.withLocked(lockDTO.lock());
        userAccount = userAccountRepository.save(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(userAccountDetailsDTO, userAccount);
    }

    @PreAuthorize("@aaaSecurityService.canDelete(#userAccountDetailsDTO, #userId)")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.USER)
    public long deleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam("userId") long userId){
        return userService.deleteUser(userId);
    }

    @PreAuthorize("@aaaSecurityService.canChangeRole(#userAccountDetailsDTO, #userId)")
    @PostMapping(Constants.Urls.API+Constants.Urls.USER + Constants.Urls.ROLE)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setRole(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam long userId, @RequestParam UserRole role){
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount = userAccount.withRole(role);
        userAccount = userAccountRepository.save(userAccount);
        return userAccountConverter.convertToUserAccountDTOExtended(userAccountDetailsDTO, userAccount);
    }

    @PreAuthorize("@aaaSecurityService.canSelfDelete(#userAccountDetailsDTO)")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.PROFILE)
    public void selfDeleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        userService.deleteUser(userId);
    }

    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.PROFILE+"/{provider}")
    public void selfDeleteBindingOauth2Provider(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @PathVariable("provider") String provider){
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
        aaaUserDetailsService.refreshUserDetails(userAccount);
    }

    @GetMapping(Constants.Urls.API + "/oauth2/providers")
    public Set<String> availableOauth2Providers() {
        return ofNullable(oAuth2ClientProperties.getIfAvailable())
                .map(OAuth2ClientProperties::getRegistration)
                .flatMap(stringRegistrationMap -> stringRegistrationMap.keySet().stream())
                .collect(Collectors.toSet());
    }

}
