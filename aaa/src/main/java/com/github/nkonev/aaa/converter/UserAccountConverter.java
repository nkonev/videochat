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

    private static SimpleGrantedAuthority convertRole(UserRole role) {
        if (role==null) {return null;}
        return new SimpleGrantedAuthority(role.name());
    }

    private static Collection<SimpleGrantedAuthority> convertRoles(Collection<UserRole> roles) {
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
                userAccount.lastLoginDateTime(),
                convertOauth(userAccount.oauth2Identifiers())
        );
    }

    public com.github.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtended(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.enabled(), userAccount.expired(), userAccount.locked(), userAccount.role());
        } else {
            dataDTO = null;
        }
        return new UserAccountDTOExtended(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
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
        userAccountDTO = validateAndTrimLogin(userAccountDTO);
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
        Assert.notNull(login, "login cannot be null");
        login = login.trim();
        Assert.hasLength(login, "login should have length");
        Assert.isTrue(!login.startsWith(FacebookOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(VkontakteOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(GoogleOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(KeycloakOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");

        return login;
    }


    private static com.github.nkonev.aaa.dto.EditUserDTO validateAndTrimLogin(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO) {
        return userAccountDTO.withLogin(validateAndTrimLogin(userAccountDTO.login()));
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
        userAccountDTO = validateAndTrimLogin(userAccountDTO);
        String password = userAccountDTO.password();
        if (!StringUtils.isEmpty(password)) {
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
        if (!StringUtils.isEmpty(userAccountDTO.email())) {
            String email = userAccountDTO.email();
            email = email.trim();
            userAccount = userAccount.withEmail(email);
        }
        return userAccount;
    }

    public static UserAccount updateUserAccountEntityNotEmpty(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        if (!StringUtils.isEmpty(userAccountDTO.login())) {
            userAccountDTO = validateAndTrimLogin(userAccountDTO);
            userAccount = userAccount.withUsername(userAccountDTO.login());
        }
        String password = userAccountDTO.password();
        if (!StringUtils.isEmpty(password)) {
            validateUserPassword(password);
            userAccount = userAccount.withPassword(passwordEncoder.encode(password));
        }
        if (Boolean.TRUE.equals(userAccountDTO.removeAvatar())){
            userAccount = userAccount.withAvatar(null);
            userAccount = userAccount.withAvatarBig(null);
        } else if (!StringUtils.isEmpty(userAccountDTO.avatar())) {
            userAccount = userAccount.withAvatar(userAccountDTO.avatar());
            userAccount = userAccount.withAvatarBig(userAccountDTO.avatarBig());
        }
        if (!StringUtils.isEmpty(userAccountDTO.email())) {
            String email = userAccountDTO.email();
            email = email.trim();
            userAccount = userAccount.withEmail(email);
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
                userAccount.avatarBig()
        );
    }

}
