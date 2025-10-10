package name.nkonev.aaa.controllers;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.*;
import name.nkonev.aaa.services.OAuth2ProvidersService;
import name.nkonev.aaa.services.UserProfileService;
import name.nkonev.aaa.services.PasswordResetService;
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

import static name.nkonev.aaa.Constants.QueryVariables.BEHALF_USER_ID;

@Controller
public class UserProfileController {

    @Autowired
    private OAuth2ProvidersService oAuth2ProvidersService;

    @Autowired
    private UserProfileService userProfileService;

    @Autowired
    private PasswordResetService passwordResetService;

    @Autowired
    private AaaProperties aaaProperties;

    private static final Logger LOGGER = LoggerFactory.getLogger(UserProfileController.class);

    /**
     *
     * @param userAccount
     * @return current logged in profile
     */
    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE, produces = MediaType.APPLICATION_JSON_VALUE)
    public name.nkonev.aaa.dto.UserSelfProfileDTO getProfile(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session) {
        LOGGER.info("Requesting external user profile");
        return userProfileService.getProfile(userAccount, session);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.checkAuthenticatedInternal(#userAccount, #requestHeaders)")
    @GetMapping(value = {Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE, Constants.Urls.INTERNAL_API + Constants.Urls.PROFILE + Constants.Urls.AUTH}, produces = MediaType.APPLICATION_JSON_VALUE)
    public HttpHeaders checkAuthenticatedInternal(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, HttpSession session, @RequestHeader HttpHeaders requestHeaders) {
        LOGGER.info("Requesting internal user profile");
        return userProfileService.processAuthenticatedInternal(userAccount, session);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER+Constants.Urls.SEARCH)
    public SearchUsersResponseDTO searchUsers(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestParam(value = "size", required = false, defaultValue = "0") int size,
            @RequestParam(value = "startingFromItemId", required = false) Long startingFromItemId,
            @RequestParam(value = "includeStartingFrom", required = false, defaultValue = "false") boolean includeStartingFrom,
            @RequestParam(value = "reverse", required = false, defaultValue = "false") boolean reverse,
            @RequestParam(value = "searchString", required = false) String searchString
    ) {
        LOGGER.info("Searching users external");
        return userProfileService.searchUsers(userAccount, new SearchUsersRequestDTO(
                size,
                startingFromItemId,
                includeStartingFrom,
                reverse,
                searchString
        ));
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @PostMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER+Constants.Urls.FRESH)
    public FreshDTO fresh(
            @RequestBody List<UserAccountDTOExtended> users,
            @RequestParam(value = "size", required = false) int size,
            @RequestParam(value = "searchString", required = false) String searchString
    ) {
        return userProfileService.freshUsers(users, size, searchString);
    }

    @ResponseBody
    @PostMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER+"/filter")
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
    @PutMapping({
        Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.REQUEST_FOR_ONLINE,
        Constants.Urls.INTERNAL_API+Constants.Urls.USER + Constants.Urls.REQUEST_FOR_ONLINE,
    })
    public void requestUserOnline(@RequestParam(value = "userId", required = false) List<Long> userIds) {
        if (userIds == null || userIds.isEmpty()) {
            return;
        }
        userProfileService.requestUserOnline(userIds);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.EXTERNAL_API +Constants.Urls.USER+Constants.Urls.USER_ID)
    public UserAccountDTOExtended getUser(
            @PathVariable(value = Constants.PathVariables.USER_ID) Long userId,
            @AuthenticationPrincipal UserAccountDetailsDTO userAccountPrincipal
    ) {
        return userProfileService.getUser(userId, userAccountPrincipal);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.EXTENDED+Constants.Urls.USER_ID)
    public UserAccountDTOExtended getUserExtendedInternal(
        @PathVariable(Constants.PathVariables.USER_ID) long userId,
        @RequestParam(BEHALF_USER_ID) long behalfUserId
    ) {
        return userProfileService.getUserExtendedInternal(userId, behalfUserId);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+Constants.Urls.LIST)
    public List<UserAccountInternalDTO> getUsersInternal(
        @RequestParam(value = "userId") List<Long> userIds
    ) {
        LOGGER.info("Getting users internal");
        return userProfileService.getUsersInternal(userIds);
    }

    @ResponseBody
    @PatchMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE)
    @PreAuthorize("isAuthenticated()")
    public UserSelfProfileDTO editNonEmpty(
            @AuthenticationPrincipal UserAccountDetailsDTO userAccount,
            @RequestBody @Valid name.nkonev.aaa.dto.EditUserDTO userAccountDTO,
            @RequestParam(defaultValue = Language.DEFAULT) Language language,
            HttpSession httpSession
    ) {
        return userProfileService.editNonEmpty(userAccount, userAccountDTO, language, httpSession);
    }

    @PreAuthorize("isAuthenticated()")
    @GetMapping(value = Constants.Urls.EXTERNAL_API + Constants.Urls.CHANGE_EMAIL_CONFIRM)
    public String changeEmailConfirm(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam(Constants.Urls.UUID) UUID uuid, HttpSession httpSession) {
        return "redirect:" + userProfileService.changeEmailConfirm(userAccount.getId(), uuid, httpSession);
    }

    @PreAuthorize("isAuthenticated()")
    @PostMapping(value = Constants.Urls.EXTERNAL_API + Constants.Urls.RESEND_CHANGE_EMAIL_CONFIRM)
    @ResponseBody
    public void resendConfirmationChangeEmailToken(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam(defaultValue = Language.DEFAULT) Language language) {
        userProfileService.resendConfirmationChangeEmailToken(userAccount, language);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.SESSIONS+"/my")
    public Map<String, Session> mySessions(@AuthenticationPrincipal UserAccountDetailsDTO userDetails){
        return userProfileService.mySessions(userDetails);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsers(@RequestParam(value = "userId") List<Long> userIds){
        return userProfileService.getOnlineForUsers(userIds);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.INTERNAL_API + Constants.Urls.USER+Constants.Urls.ONLINE)
    public List<UserOnlineResponse> getOnlineForUsersInternal(@RequestParam(value = "userId") List<Long> userIds){
        return userProfileService.getOnlineForUsersInternal(userIds);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.hasSessionManagementPermission(#userAccount)")
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.SESSIONS)
    public Map<String, Session> sessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId){
        return userProfileService.sessions(userAccount, userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canRemoveSessions(#userAccount, #userId)")
    @DeleteMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.SESSIONS)
    public void killSessions(@AuthenticationPrincipal UserAccountDetailsDTO userAccount, @RequestParam("userId") long userId, HttpSession httpSession){
        userProfileService.killSessions(userAccount, userId, httpSession);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canLock(#userAccountDetailsDTO, #lockDTO)")
    @PostMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.LOCK)
    public name.nkonev.aaa.dto.UserAccountDTOExtended setLocked(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody name.nkonev.aaa.dto.LockDTO lockDTO){
        return userProfileService.setLocked(userAccountDetailsDTO, lockDTO);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canConfirm(#userAccountDetailsDTO, #confirmDTO)")
    @PostMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.CONFIRM)
    public name.nkonev.aaa.dto.UserAccountDTOExtended setConfirmed(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody name.nkonev.aaa.dto.ConfirmDTO confirmDTO){
        return userProfileService.setConfirmed(userAccountDetailsDTO, confirmDTO);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canEnable(#userAccountDetailsDTO, #enableDTO)")
    @PostMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.ENABLE)
    public name.nkonev.aaa.dto.UserAccountDTOExtended setEnabled(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody name.nkonev.aaa.dto.EnabledDTO enableDTO){
        return userProfileService.setEnabled(userAccountDetailsDTO, enableDTO);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canDelete(#userAccountDetailsDTO, #userId)")
    @DeleteMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER)
    public void deleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestParam("userId") long userId){
        userProfileService.deleteUser(userAccountDetailsDTO, userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canChangeRole(#userAccountDetailsDTO, #setRolesDTO.userId)")
    @PutMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.ROLE)
    public name.nkonev.aaa.dto.UserAccountDTOExtended setRoles(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody SetRolesDTO setRolesDTO){
        return userProfileService.setRoles(userAccountDetailsDTO, setRolesDTO.userId(), setRolesDTO.roles());
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canChangePermissions(#userAccountDetailsDTO, #setOverriddenPermissionsDTO.userId)")
    @PutMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.PERMISSION)
    public name.nkonev.aaa.dto.UserAccountDTOExtended setPermissions(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO, @RequestBody SetOverriddenPermissionsDTO setOverriddenPermissionsDTO){
        return userProfileService.setPermissions(userAccountDetailsDTO, setOverriddenPermissionsDTO.userId(), setOverriddenPermissionsDTO.addPermissions(), setOverriddenPermissionsDTO.removePermissions());
    }

    @ResponseBody
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.ROLE)
    public List<UserRole> getAllRoles() {
        return Arrays.stream(UserRole.values()).toList();
    }

    @ResponseBody
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.PERMISSION)
    public List<ExternalPermission> getAllPermissions() {
        return Arrays.stream(ExternalPermission.values()).toList();
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canChangePermissions(#userAccountDetailsDTO, #userId)")
    @GetMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.USER + Constants.Urls.PERMISSION + Constants.Urls.USER_ID)
    public OverriddenPermissionsDTO getUserOverriddenPermissions(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO,
         @PathVariable(value = Constants.PathVariables.USER_ID) Long userId
    ) {
        return userProfileService.getUserOverriddenPermissions(userId);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canSelfDelete(#userAccountDetailsDTO)")
    @DeleteMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE)
    public void selfDeleteUser(@AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO){
        userProfileService.selfDeleteUser(userAccountDetailsDTO);
    }

    @ResponseBody
    @PreAuthorize("isAuthenticated()")
    @DeleteMapping(Constants.Urls.EXTERNAL_API +Constants.Urls.PROFILE+"/{provider}")
    public UserSelfProfileDTO selfDeleteBindingOauth2Provider(
        @AuthenticationPrincipal UserAccountDetailsDTO userAccountDetailsDTO,
        @PathVariable("provider") String provider,
        HttpSession httpSession
    ){
        return userProfileService.selfDeleteBindingOauth2Provider(userAccountDetailsDTO, provider, httpSession);
    }

    @ResponseBody
    @GetMapping(Constants.Urls.EXTERNAL_API + Constants.Urls.CONFIG)
    public ConfigDTO aaaConfig() {
        return new ConfigDTO(oAuth2ProvidersService.availableOauth2Providers(), aaaProperties.frontendSessionPingInterval().toMillis(), Constants.MIN_PASSWORD_LENGTH, Constants.MAX_PASSWORD_LENGTH);
    }

    @ResponseBody
    @GetMapping(value = Constants.Urls.INTERNAL_API+Constants.Urls.USER+"/exist")
    public List<UserExists> getUsersExistInternal(
        @RequestParam(value = "userId") List<Long> requestedUserIds
    ) {
        LOGGER.info("Requesting internal users exist {}", requestedUserIds);

        return userProfileService.getUsersExistInternal(requestedUserIds);
    }

    @ResponseBody
    @PreAuthorize("@aaaInternalPermissionService.canSetPassword(#userAccount, #userId)")
    @PutMapping(Constants.Urls.EXTERNAL_API + Constants.Urls.USER+Constants.Urls.USER_ID + Constants.Urls.PASSWORD)
    public void setPassword(@AuthenticationPrincipal UserAccountDetailsDTO userAccount,
                            @PathVariable(value = Constants.PathVariables.USER_ID) Long userId,
                            @RequestBody @Valid SetPasswordDTO setPasswordDTO){
        passwordResetService.setPassword(setPasswordDTO, userId);
    }

    @ResponseBody
    @PutMapping(Constants.Urls.EXTERNAL_API + Constants.Urls.PING)
    public Map<String, Boolean> pingSession(@AuthenticationPrincipal UserAccountDetailsDTO userAccount){
        return Map.of("status", userAccount != null);
    }
}
