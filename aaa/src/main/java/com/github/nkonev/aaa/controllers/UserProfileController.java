package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.converter.UserAccountConverter;
import com.github.nkonev.aaa.dto.EditUserDTO;
import com.github.nkonev.aaa.dto.UserAccountDTO;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.exception.UserAlreadyPresentException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaUserDetailsService;
import com.github.nkonev.aaa.services.UserDeleteService;
import com.github.nkonev.aaa.utils.PageUtils;
import name.nkonev.aaa.UserDto;
import name.nkonev.aaa.UsersRequest;
import name.nkonev.aaa.UsersResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
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
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.Constants.Headers.*;
import static com.github.nkonev.aaa.Constants.MAX_USERS_RESPONSE_LENGTH;
import static com.github.nkonev.aaa.converter.UserAccountConverter.convertRolesToStringList;

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
    private UserDeleteService userDeleteService;

    public static final String X_PROTOBUF_CHARSET_UTF_8 = "application/x-protobuf;charset=UTF-8";

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
        Long expiresAt = getExpiresAt(session);
        return UserAccountConverter.getUserSelfProfile(userAccount, userAccount.getLastLoginDateTime(), expiresAt);
    }

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE)
    public HttpHeaders checkAuthenticatedInternal(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting internal user profile");
        Long expiresAt = getExpiresAt(session);
        var dto = checkAuthenticated(userAccount, session);
        HttpHeaders headers = new HttpHeaders();
        headers.set(X_AUTH_USERNAME, dto.getLogin());
        headers.set(X_AUTH_USER_ID, ""+userAccount.getId());
        headers.set(X_AUTH_EXPIRESIN, ""+expiresAt);
        convertRolesToStringList(userAccount.getRoles()).forEach(s -> {
            headers.add(X_AUTH_ROLE, s);
        });
        return headers;
    }


    @GetMapping(value = Constants.Urls.API+Constants.Urls.USER)
    public com.github.nkonev.aaa.dto.Wrapper<com.github.nkonev.aaa.dto.UserAccountDTO> getUsers(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestParam(value = "page", required=false, defaultValue = "0") int page,
            @RequestParam(value = "size", required=false, defaultValue = "0") int size,
            @RequestParam(value = "searchString", required=false, defaultValue = "") String searchString
    ) {
        PageRequest springDataPage = PageRequest.of(PageUtils.fixPage(page), PageUtils.fixSize(size), Sort.Direction.ASC, "id");
        searchString = searchString.trim();

        final String forDbSearch = "%" + searchString + "%";
        List<UserAccount> resultPage = userAccountRepository.findByUsernameContainsIgnoreCase(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch);
        long resultPageCount = userAccountRepository.findByUsernameContainsIgnoreCaseCount(springDataPage.getPageSize(), springDataPage.getOffset(), forDbSearch);

        return new com.github.nkonev.aaa.dto.Wrapper<com.github.nkonev.aaa.dto.UserAccountDTO>(
                resultPage.stream().map(getConvertToUserAccountDTO(userAccount)).collect(Collectors.toList()),
                resultPageCount
        );
    }

    private Function<UserAccount, com.github.nkonev.aaa.dto.UserAccountDTO> getConvertToUserAccountDTO(UserAccountDetailsDTO currentUser) {
        return userAccount -> userAccountConverter.convertToUserAccountDTOExtended(currentUser, userAccount);
    }

    @GetMapping(value = Constants.Urls.API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<com.github.nkonev.aaa.dto.UserAccountDTO> getUsers(
            @RequestParam(value = "userId") List<Long> userIds,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
        ) {
        if (userIds == null) {
            throw new BadRequestException("Cannot be null");
        }
        if (userIds.size() > MAX_USERS_RESPONSE_LENGTH) {
            throw new BadRequestException("Cannot be greater than " + MAX_USERS_RESPONSE_LENGTH);
        }
        List<com.github.nkonev.aaa.dto.UserAccountDTO> result = new ArrayList<>();
        for (UserAccount userAccountEntity: userAccountRepository.findByIdInOrderById(userIds)) {
            if (userAccountPrincipal != null && userAccountPrincipal.getId().equals(userAccountEntity.getId())) {
                result.add(UserAccountConverter.getUserSelfProfile(userAccountPrincipal, userAccountEntity.getLastLoginDateTime(), null));
            } else {
                result.add(userAccountConverter.convertToUserAccountDTO(userAccountEntity));
            }
        }
        return result;
    }

    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.LIST, produces = X_PROTOBUF_CHARSET_UTF_8)
    public UsersResponse getUserInternal(@RequestBody UsersRequest usersRequest,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        LOGGER.info("Requesting internal users {}", usersRequest);
        List<UserAccountDTO> users = getUsers(usersRequest.getUserIdsList(), userAccountPrincipal);
        UsersResponse.Builder responseBuilder = UsersResponse.newBuilder();

        users.forEach(userAccountDTO -> {
            UserDto.Builder builder = UserDto.newBuilder()
                    .setId(userAccountDTO.getId())
                    .setLogin(userAccountDTO.getLogin());
            if (userAccountDTO.getAvatar() != null) {
                builder.setAvatar(userAccountDTO.getAvatar());
            }
            UserDto user = builder.build();
            responseBuilder.addUsers(user);
        });

        return responseBuilder.build();
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

        // check email already present
        if (!checkEmail(userAccountDTO, exists)) return UserAccountConverter.convertToEditUserDto(exists);

        // check login already present
        checkLogin(userAccountDTO, exists);

        UserAccountConverter.updateUserAccountEntity(userAccountDTO, exists, passwordEncoder);
        exists = userAccountRepository.save(exists);

        aaaUserDetailsService.refreshUserDetails(exists);

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
        if (!checkEmail(userAccountDTO, exists))
            return UserAccountConverter.convertToEditUserDto(exists); // we care for email leak...

        // check login already present
        checkLogin(userAccountDTO, exists);

        UserAccountConverter.updateUserAccountEntityNotEmpty(userAccountDTO, exists, passwordEncoder);
        exists = userAccountRepository.save(exists);

        aaaUserDetailsService.refreshUserDetails(exists);

        return UserAccountConverter.convertToEditUserDto(exists);
    }

    private void checkLogin(@RequestBody @Valid EditUserDTO userAccountDTO, UserAccount exists) {
        if (!exists.getUsername().equals(userAccountDTO.getLogin()) && userAccountRepository.findByUsername(userAccountDTO.getLogin()).isPresent()) {
            throw new UserAlreadyPresentException("User with login '" + userAccountDTO.getLogin() + "' is already present");
        }
    }

    private boolean checkEmail(@RequestBody @Valid EditUserDTO userAccountDTO, UserAccount exists) {
        if (exists.getEmail() != null && !exists.getEmail().equals(userAccountDTO.getEmail()) && userAccountRepository.findByEmail(userAccountDTO.getEmail()).isPresent()) {
            LOGGER.error("editProfile: user with email '{}' already present. exiting...", exists.getEmail());
            return false;
        } else {
            return true;
        }
    }


    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.API+Constants.Urls.SESSIONS+"/my")
    public Map<String, Session> mySessions(@AuthenticationPrincipal UserAccountDetailsDTO userDetails){
        return aaaUserDetailsService.getMySessions(userDetails);
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
        UserAccount userAccount = aaaUserDetailsService.getUserAccount(lockDTO.getUserId());
        if (lockDTO.isLock()){
            aaaUserDetailsService.killSessions(lockDTO.getUserId());
        }
        userAccount.setLocked(lockDTO.isLock());
        userAccount = userAccountRepository.save(userAccount);

        return userAccountConverter.convertToUserAccountDTOExtended(userAccountDetailsDTO, userAccount);
    }

    @PreAuthorize("@aaaSecurityService.canDelete(#userAccountDetailsDTO, #userId)")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.USER)
    public long deleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam("userId") long userId){
        return userDeleteService.deleteUser(userId);
    }

    @PreAuthorize("@aaaSecurityService.canChangeRole(#userAccountDetailsDTO, #userId)")
    @PostMapping(Constants.Urls.API+Constants.Urls.USER + Constants.Urls.ROLE)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setRole(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam long userId, @RequestParam UserRole role){
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount.setRole(role);
        userAccount = userAccountRepository.save(userAccount);
        return userAccountConverter.convertToUserAccountDTOExtended(userAccountDetailsDTO, userAccount);
    }

    @PreAuthorize("@aaaSecurityService.canSelfDelete(#userAccountDetailsDTO)")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.PROFILE)
    public void selfDeleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        userDeleteService.deleteUser(userId);
    }

    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.PROFILE+Constants.Urls.FACEBOOK)
    public void selfDeleteBindingFacebook(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount.getOauthIdentifiers().setFacebookId(null);
        userAccount = userAccountRepository.save(userAccount);
        aaaUserDetailsService.refreshUserDetails(userAccount);
    }

    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.API+Constants.Urls.PROFILE+Constants.Urls.VKONTAKTE)
    public void selfDeleteBindingVkontakte(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        long userId = userAccountDetailsDTO.getId();
        UserAccount userAccount = userAccountRepository.findById(userId).orElseThrow();
        userAccount.getOauthIdentifiers().setVkontakteId(null);
        userAccount = userAccountRepository.save(userAccount);
        aaaUserDetailsService.refreshUserDetails(userAccount);
    }

}