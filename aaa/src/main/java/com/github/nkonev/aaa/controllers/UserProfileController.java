package com.github.nkonev.aaa.controllers;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.*;
import com.github.nkonev.aaa.services.OAuth2ProvidersService;
import com.github.nkonev.aaa.services.UserProfileService;
import jakarta.servlet.http.HttpSession;
import jakarta.validation.Valid;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.session.Session;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.*;

import java.util.*;

/**
 * Created by nik on 08.06.17.
 */
@Controller
public class UserProfileController {

    @Autowired
    private OAuth2ProvidersService oAuth2ProvidersService;

    @Autowired
    private UserProfileService userProfileService;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserProfileController.class);

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
        return userProfileService.checkAuthenticated(userAccount, session);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = {Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE, Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH}, produces = MediaType.APPLICATION_JSON_VALUE)
    public HttpHeaders checkAuthenticatedInternal(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting internal user profile");
        return userProfileService.checkAuthenticatedInternal(userAccount, session);
    }

    @ResponseBody
    @CrossOrigin(origins="*", methods = RequestMethod.POST)
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.SEARCH)
    public List<com.github.nkonev.aaa.dto.UserAccountDTOExtended> searchUsers(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody SearchUsersRequestDTO request
    ) {
        LOGGER.info("Searching users external");
        return userProfileService.searchUsers(userAccount, request);
    }

    @ResponseBody
    @PutMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER+"/filter")
    public Map<String, Boolean> filter(@RequestBody FilterUserRequest filterUserRequest) {
        return userProfileService.filter(filterUserRequest);
    }

    @ResponseBody
    @CrossOrigin(origins="*", methods = RequestMethod.POST)
    @PostMapping(Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.SEARCH)
    public SearchUsersResponseInternalDTO searchUsersInternal(@RequestBody SearchUsersRequestInternalDTO request) {
        LOGGER.info("Searching users internal");
        return userProfileService.searchUsersInternal(request);
    }

    @ResponseBody
    @PutMapping(Constants.Urls.INTERNAL_API+Constants.Urls.USER + Constants.Urls.REQUEST_FOR_ONLINE)
    public void requestUserOnline(@RequestParam(value = "userId") List<Long> userIds) {
        userProfileService.requestUserOnline(userIds);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.USER_ID)
    public Record getUser(
            @PathVariable(value = Constants.PathVariables.USER_ID) Long userId,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        return userProfileService.getUser(userId, userAccountPrincipal);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<UserAccountDTO> getUsersInternal(
        @RequestParam(value = "userId") List<Long> userIds
    ) {
        return userProfileService.getUsersInternal(userIds);
    }

    @ResponseBody
    @PatchMapping(Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE)
    @PreAuthorize("isAuthenticated()")
    public UserSelfProfileDTO editNonEmpty(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody @Valid com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO,
            @RequestParam(defaultValue = Language.DEFAULT) Language language,
            HttpSession httpSession
    ) {
        return userProfileService.editNonEmpty(userAccount, userAccountDTO, language, httpSession);
    }

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.CHANGE_EMAIL_CONFIRM)
    public String changeEmailConfirm(@RequestParam(Constants.Urls.UUID) UUID uuid, HttpSession httpSession) {
        return userProfileService.changeEmailConfirm(uuid, httpSession);
    }

    @PreAuthorize("isAuthenticated()")
    @PostMapping(value = Constants.Urls.PUBLIC_API + Constants.Urls.RESEND_CHANGE_EMAIL_CONFIRM)
    @ResponseBody
    public void resendConfirmationChangeEmailToken(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        userProfileService.resendConfirmationChangeEmailToken(userAccount, language);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS+"/my")
    public Map<String, Session> mySessions(@AuthenticationPrincipal UserAccountDetailsDTO userDetails){
        return userProfileService.mySessions(userDetails);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsers(@RequestParam(value = "userId") List<Long> userIds){
        return userProfileService.getOnlineForUsers(userIds);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.INTERNAL_API + Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsersInternal(@RequestParam(value = "userId") List<Long> userIds){
        return userProfileService.getOnlineForUsersInternal(userIds);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.hasSessionManagementPermission(#userAccount)")
    @GetMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS)
    public Map<String, Session> sessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        return userProfileService.sessions(userAccount, userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.hasSessionManagementPermission(#userAccount)")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.SESSIONS)
    public void killSessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        userProfileService.killSessions(userAccount, userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canLock(#userAccountDetailsDTO, #lockDTO)")
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.LOCK)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setLocked(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody com.github.nkonev.aaa.dto.LockDTO lockDTO){
        return userProfileService.setLocked(userAccountDetailsDTO, lockDTO);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canConfirm(#userAccountDetailsDTO, #confirmDTO)")
    @PostMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.CONFIRM)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setConfirmed(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody com.github.nkonev.aaa.dto.ConfirmDTO confirmDTO){
        return userProfileService.setNewEmailConfirmed(userAccountDetailsDTO, confirmDTO);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canDelete(#userAccountDetailsDTO, #userId)")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER)
    public void deleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam("userId") long userId){
        userProfileService.deleteUser(userAccountDetailsDTO, userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaPermissionService.canChangeRole(#userAccountDetailsDTO, #userId)")
    @PutMapping(Constants.Urls.PUBLIC_API +Constants.Urls.USER + Constants.Urls.ROLE)
    public com.github.nkonev.aaa.dto.UserAccountDTOExtended setRole(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam long userId, @RequestParam UserRole role){
        return userProfileService.setRole(userAccountDetailsDTO, userId, role);
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
        userProfileService.selfDeleteUser(userAccountDetailsDTO);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.PUBLIC_API +Constants.Urls.PROFILE+"/{provider}")
    public UserSelfProfileDTO selfDeleteBindingOauth2Provider(
        @AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO,
        @PathVariable("provider") String provider,
        HttpSession httpSession
    ){
        return userProfileService.selfDeleteBindingOauth2Provider(userAccountDetailsDTO, provider, httpSession);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.PUBLIC_API + "/oauth2/providers")
    public Set<String> availableOauth2Providers() {
        return oAuth2ProvidersService.availableOauth2Providers();
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+"/exist")
    public List<UserExists> getUsersExistInternal(
        @RequestParam(value = "userId") List<Long> requestedUserIds
    ) {
        LOGGER.info("Requesting internal users exist {}", requestedUserIds);

        return userProfileService.getUsersExistInternal(requestedUserIds);
    }
}
