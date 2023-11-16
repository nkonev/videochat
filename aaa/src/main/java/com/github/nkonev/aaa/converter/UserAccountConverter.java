package com.github.nkonev.aaa.converter;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.OAuth2IdentifiersDTO;
import com.github.nkonev.aaa.dto.UserAccountDTOExtended;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.CreationType;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.security.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;
import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

import static com.github.nkonev.aaa.Constants.FORBIDDEN_USERNAMES;

@Component
public class UserAccountConverter {

    @Autowired
    private AaaSecurityService aaaSecurityService;

    private static UserRole getDefaultUserRole(){
        return UserRole.ROLE_USER;
    }

    public static List<String> convertRolesToStringList(Collection<GrantedAuthority> roles) {
        return Optional.ofNullable(roles).map(rs -> rs.stream().map(GrantedAuthority::getAuthority).collect(Collectors.toList())).orElse(Collections.emptyList());
    }

    private static OAuth2IdentifiersDTO convertOauth(UserAccount.OAuth2Identifiers oAuth2Identifiers){
        if (oAuth2Identifiers ==null) return null;
        return new OAuth2IdentifiersDTO(oAuth2Identifiers.facebookId(), oAuth2Identifiers.vkontakteId(), oAuth2Identifiers.googleId(), oAuth2Identifiers.keycloakId());
    }

    public static UserAccountDetailsDTO convertToUserAccountDetailsDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        return new UserAccountDetailsDTO(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                userAccount.password(),
                userAccount.expired(),
                userAccount.locked(),
                userAccount.enabled(),
                Collections.singletonList(convertRole(userAccount.role())),
                userAccount.email(),
                userAccount.lastLoginDateTime(),
                convertOauth(userAccount.oauth2Identifiers())
        );
    }

    public static com.github.nkonev.aaa.dto.UserSelfProfileDTO getUserSelfProfile(UserAccountDetailsDTO userAccount, LocalDateTime lastLoginDateTime, Long expiresAt) {
        if (userAccount == null) { return null; }
        return new com.github.nkonev.aaa.dto.UserSelfProfileDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getAvatarBig(),
                userAccount.userAccountDTO().shortInfo(),
                userAccount.getEmail(),
                lastLoginDateTime,
                userAccount.getOauth2Identifiers(),
                convertRoles2Enum(userAccount.getRoles()),
                expiresAt
        );
    }

    private static Collection<UserRole> convertRoles2Enum(Collection<GrantedAuthority> roles) {
        if (roles == null) {
            return null;
        } else {
            return roles.stream().map(grantedAuthority -> UserRole.valueOf(grantedAuthority.getAuthority())).collect(Collectors.toList());
        }
    }

    public static SimpleGrantedAuthority convertRole(UserRole role) {
        if (role==null) {return null;}
        return new SimpleGrantedAuthority(role.name());
    }

    public static Collection<SimpleGrantedAuthority> convertRoles(Collection<UserRole> roles) {
        if (roles==null) {return null;}
        return roles.stream().map(ur -> new SimpleGrantedAuthority(ur.name())).collect(Collectors.toSet());
    }

    public static com.github.nkonev.aaa.dto.UserAccountDTO convertToUserAccountDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        return new com.github.nkonev.aaa.dto.UserAccountDTO(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                userAccount.lastLoginDateTime(),
                convertOauth(userAccount.oauth2Identifiers())
        );
    }

    public com.github.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtended(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.enabled(), userAccount.expired(), userAccount.locked(), Set.of(userAccount.role()));
        } else {
            dataDTO = null;
        }
        return new UserAccountDTOExtended(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                dataDTO,
                userAccount.lastLoginDateTime(),
                convertOauth(userAccount.oauth2Identifiers()),
                aaaSecurityService.canLock(currentUser, userAccount),
                aaaSecurityService.canDelete(currentUser, userAccount),
                aaaSecurityService.canChangeRole(currentUser, userAccount)
        );
    }

    public com.github.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtendedForAdmin(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermissionForAdmin()){
            dataDTO = new com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.enabled(), userAccount.expired(), userAccount.locked(), Set.of(userAccount.role()));
        } else {
            dataDTO = null;
        }
        return new UserAccountDTOExtended(
            userAccount.id(),
            userAccount.username(),
            userAccount.avatar(),
            userAccount.avatarBig(),
            userAccount.shortInfo(),
            dataDTO,
            userAccount.lastLoginDateTime(),
            convertOauth(userAccount.oauth2Identifiers()),
            aaaSecurityService.canLock(currentUser, userAccount),
            aaaSecurityService.canDelete(currentUser, userAccount),
            aaaSecurityService.canChangeRole(currentUser, userAccount)
        );
    }


    private static void validateUserPassword(String password) {
        Assert.notNull(password, "password must be set");
        if (password.length() < Constants.MIN_PASSWORD_LENGTH || password.length() > Constants.MAX_PASSWORD_LENGTH) {
            throw new BadRequestException("password don't match requirements");
        }
    }

    public static UserAccount buildUserAccountEntityForInsert(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, PasswordEncoder passwordEncoder) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = false;

        final UserRole newUserRole = getDefaultUserRole();

        validateLoginAndEmail(userAccountDTO);
        userAccountDTO = trimAndValidateNonAouth2Login(userAccountDTO);
        String password = userAccountDTO.password();
        try {
            validateUserPassword(password);
        } catch (IllegalArgumentException e) {
            throw new BadRequestException(e.getMessage());
        }

        return new UserAccount(
                null,
                CreationType.REGISTRATION,
                userAccountDTO.login(),
                passwordEncoder.encode(password),
                userAccountDTO.avatar(),
                userAccountDTO.avatarBig(),
                userAccountDTO.shortInfo(),
                expired,
                locked,
                enabled,
                newUserRole,
                userAccountDTO.email(),
                null,
                null
        );
    }

    public static String validateAndTrimLogin(String login){
        login = login != null ? login.trim() : null;

        if (!StringUtils.hasLength(login)) {
            throw new BadRequestException("empty login");
        }
        if (FORBIDDEN_USERNAMES.contains(login)) {
            throw new BadRequestException("forbidden login");
        }

        return login;
    }

    public static void validateLoginNonAouth2(String login){
        Assert.isTrue(!login.startsWith(FacebookOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(VkontakteOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(GoogleOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(KeycloakOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
    }


    public static com.github.nkonev.aaa.dto.EditUserDTO trimAndValidateNonAouth2Login(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO) {
        var ret = userAccountDTO.withLogin(validateAndTrimLogin(userAccountDTO.login()));
        validateLoginNonAouth2(ret.login());
        return ret;
    }

    // used for just get user id
    public static UserAccount buildUserAccountEntityForFacebookInsert(String facebookId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.FACEBOOK,
                login,
                null,
                maybeImageUrl,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                null,
                new UserAccount.OAuth2Identifiers(facebookId, null, null, null)
        );
    }

    public static UserAccount buildUserAccountEntityForVkontakteInsert(String vkontakteId, String login) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.VKONTAKTE,
                login,
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                null,
                new UserAccount.OAuth2Identifiers(null, vkontakteId, null, null)
        );
    }

    public static UserAccount buildUserAccountEntityForGoogleInsert(String googleId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.GOOGLE,
                login,
                null,
                maybeImageUrl,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                null,
                new UserAccount.OAuth2Identifiers(null, null, googleId, null)
        );
    }

    public static UserAccount buildUserAccountEntityForKeycloakInsert(String keycloakId, String login, String maybeImageUrl, boolean hasAdminRole) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = hasAdminRole ? UserRole.ROLE_ADMIN : getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.KEYCLOAK,
                login,
                null,
                maybeImageUrl,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                null,
                new UserAccount.OAuth2Identifiers(null, null, null, keycloakId)
        );
    }

    public static UserAccount buildUserAccountEntityForLdapInsert(String login) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.LDAP,
                login,
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                null,
                new UserAccount.OAuth2Identifiers(null, null, null, null)
        );
    }

    private static void validateLoginAndEmail(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO){
        Assert.hasLength(userAccountDTO.login(), "login should have length");
        Assert.hasLength(userAccountDTO.email(), "email should have length");
    }

    public static UserAccount updateUserAccountEntity(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        Assert.hasLength(userAccountDTO.login(), "login should have length");
        userAccountDTO = trimAndValidateNonAouth2Login(userAccountDTO);
        String password = userAccountDTO.password();
        if (StringUtils.hasLength(password)) {
            validateUserPassword(password);
            userAccount = userAccount.withPassword(passwordEncoder.encode(password));
        }
        userAccount = userAccount.withUsername(userAccountDTO.login());
        if (Boolean.TRUE.equals(userAccountDTO.removeAvatar())){
            userAccount = userAccount.withAvatar(null);
            userAccount = userAccount.withAvatarBig(null);
        } else {
            userAccount = userAccount.withAvatar(userAccountDTO.avatar());
            userAccount = userAccount.withAvatarBig(userAccountDTO.avatarBig());
        }
        if (StringUtils.hasLength(userAccountDTO.email())) {
            String email = userAccountDTO.email();
            email = email.trim();
            userAccount = userAccount.withEmail(email);
        }
        if (StringUtils.hasLength(userAccountDTO.shortInfo())) {
            userAccount = userAccount.withShortInfo(userAccountDTO.shortInfo());
        } else {
            userAccount = userAccount.withShortInfo(null);
        }
        return userAccount;
    }

    public static UserAccount updateUserAccountEntityNotEmpty(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        if (StringUtils.hasLength(userAccountDTO.login())) {
            userAccountDTO = trimAndValidateNonAouth2Login(userAccountDTO);
            userAccount = userAccount.withUsername(userAccountDTO.login());
        }
        String password = userAccountDTO.password();
        if (StringUtils.hasLength(password)) {
            validateUserPassword(password);
            userAccount = userAccount.withPassword(passwordEncoder.encode(password));
        }
        if (Boolean.TRUE.equals(userAccountDTO.removeAvatar())){
            userAccount = userAccount.withAvatar(null);
            userAccount = userAccount.withAvatarBig(null);
        } else if (StringUtils.hasLength(userAccountDTO.avatar())) {
            userAccount = userAccount.withAvatar(userAccountDTO.avatar());
            userAccount = userAccount.withAvatarBig(userAccountDTO.avatarBig());
        }
        if (StringUtils.hasLength(userAccountDTO.email())) {
            String email = userAccountDTO.email();
            email = email.trim();
            userAccount = userAccount.withEmail(email);
        }
        if (StringUtils.hasLength(userAccountDTO.shortInfo())) {
            userAccount = userAccount.withShortInfo(userAccountDTO.shortInfo());
        }

        return userAccount;
    }

    public static com.github.nkonev.aaa.dto.EditUserDTO convertToEditUserDto(UserAccount userAccount) {
        return new com.github.nkonev.aaa.dto.EditUserDTO(
                userAccount.username(),
                userAccount.avatar(),
                null,
                null,
                userAccount.email(),
                userAccount.avatarBig(),
                userAccount.shortInfo()
        );
    }

}
