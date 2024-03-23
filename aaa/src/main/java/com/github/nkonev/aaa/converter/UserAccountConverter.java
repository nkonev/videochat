package com.github.nkonev.aaa.converter;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.*;
import com.github.nkonev.aaa.entity.jdbc.CreationType;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.security.*;
import com.github.nkonev.aaa.utils.NullEncode;
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
    private AaaPermissionService aaaSecurityService;

    private static UserRole getDefaultUserRole(){
        return UserRole.ROLE_USER;
    }

    public static EditUserDTO normalize(EditUserDTO editUserDTO, boolean isForOauth2) {
        var userAccountDTO = editUserDTO.withLogin(checkAndTrimLogin(editUserDTO.login(), isForOauth2));
        userAccountDTO = userAccountDTO.withEmail(trimToNull(NullEncode.forHtml(userAccountDTO.email())));
        userAccountDTO = userAccountDTO.withAvatar(trimToNull(NullEncode.forHtmlAttribute(userAccountDTO.avatar())));
        userAccountDTO = userAccountDTO.withAvatarBig(trimToNull(NullEncode.forHtmlAttribute(userAccountDTO.avatarBig())));
        userAccountDTO = userAccountDTO.withShortInfo(trimToNull(NullEncode.forHtml(userAccountDTO.shortInfo())));
        userAccountDTO = userAccountDTO.withLoginColor(trimToNull(userAccountDTO.loginColor()));
        return userAccountDTO;
    }

    public static List<String> convertRolesToStringList(Collection<GrantedAuthority> roles) {
        return Optional.ofNullable(roles).map(rs -> rs.stream().map(GrantedAuthority::getAuthority).collect(Collectors.toList())).orElse(Collections.emptyList());
    }

    private static OAuth2IdentifiersDTO convertOAuth2(UserAccount.OAuth2Identifiers oAuth2Identifiers){
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
                userAccount.confirmed(),
                Collections.singletonList(convertRole(userAccount.role())),
                userAccount.email(),
                StringUtils.hasLength(userAccount.newEmail()),
                userAccount.lastLoginDateTime(),
                convertOAuth2(userAccount.oauth2Identifiers()),
                userAccount.loginColor()
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
                userAccount.awaitingForConfirmEmailChange(),
                lastLoginDateTime,
                userAccount.getOauth2Identifiers(),
                convertRoles2Enum(userAccount.getRoles()),
                expiresAt,
                userAccount.getLoginColor()
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
                convertOAuth2(userAccount.oauth2Identifiers()),
                userAccount.loginColor()
        );
    }

    public com.github.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtended(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.enabled(), userAccount.expired(), userAccount.locked(), userAccount.confirmed(), Set.of(userAccount.role()));
        } else {
            dataDTO = null;
        }
        var awaitingForConfirmEmailChange = StringUtils.hasLength(userAccount.newEmail());
        return new UserAccountDTOExtended(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                dataDTO,
                userAccount.lastLoginDateTime(),
                convertOAuth2(userAccount.oauth2Identifiers()),
                aaaSecurityService.canLock(currentUser, userAccount),
                aaaSecurityService.canDelete(currentUser, userAccount),
                aaaSecurityService.canChangeRole(currentUser, userAccount),
                aaaSecurityService.canConfirm(currentUser, userAccount),
                awaitingForConfirmEmailChange,
                userAccount.loginColor()
        );
    }

    private static void validateUserPassword(String password) {
        Assert.notNull(password, "password must be set");
        if (password.length() < Constants.MIN_PASSWORD_LENGTH || password.length() > Constants.MAX_PASSWORD_LENGTH) {
            throw new BadRequestException("password don't match requirements");
        }
    }

    // EditUserDTO userAccountDTO is already filtered by normalize()
    public static UserAccount buildUserAccountEntityForInsert(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, PasswordEncoder passwordEncoder) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = false;

        final UserRole newUserRole = getDefaultUserRole();

        validateLoginAndEmail(userAccountDTO);
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
                confirmed,
                newUserRole,
                userAccountDTO.email(),
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                userAccountDTO.loginColor()
        );
    }

    public static String validateLengthAndTrimLogin(String login, boolean isForOauth2) {
        login = checkAndTrimLogin(login, isForOauth2);

        if (!StringUtils.hasLength(login)) {
            throw new BadRequestException("login must be set");
        }

        return login;
    }

    public static void validateLengthEmail(String email) {
        if (!StringUtils.hasLength(email)) {
            throw new BadRequestException("email must be set");
        }
    }

    private static String checkAndTrimLogin(String login, boolean isForOauth2) {
        login = login != null ? login.trim() : null;
        login = trimToNull(login);

        if (login != null) {
            if (FORBIDDEN_USERNAMES.contains(login)) {
                throw new BadRequestException("forbidden login");
            }
        }

        login = NullEncode.forHtml(login);

        if (login != null && !isForOauth2) {
            Assert.isTrue(!login.startsWith(FacebookOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
            Assert.isTrue(!login.startsWith(VkontakteOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
            Assert.isTrue(!login.startsWith(GoogleOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
            Assert.isTrue(!login.startsWith(KeycloakOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        }

        return login;
    }

    // used for just get user id
    public static UserAccount buildUserAccountEntityForFacebookInsert(String facebookId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.FACEBOOK,
                NullEncode.forHtml(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRole,
                null,
                null,
                null,
                facebookId,
                null,
                null,
                null,
                null,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForVkontakteInsert(String vkontakteId, String login) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.VKONTAKTE,
                NullEncode.forHtml(login),
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRole,
                null,
                null,
                null,
                null,
                vkontakteId,
                null,
                null,
                null,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForGoogleInsert(String googleId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.GOOGLE,
                NullEncode.forHtml(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRole,
                null,
                null,
                null,
                null,
                null,
                googleId,
                null,
                null,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForKeycloakInsert(String keycloakId, String login, String maybeImageUrl, boolean hasAdminRole) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        final UserRole newUserRole = hasAdminRole ? UserRole.ROLE_ADMIN : getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.KEYCLOAK,
                NullEncode.forHtml(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRole,
                null,
                null,
                null,
                null,
                null,
                null,
                keycloakId,
                null,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForLdapInsert(String login, String ldapId) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                null,
                CreationType.LDAP,
                NullEncode.forHtml(login),
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRole,
                null,
                null,
                null,
                null,
                null,
                null,
                null,
                ldapId,
                null
        );
    }

    private static void validateLoginAndEmail(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO){
        Assert.hasLength(userAccountDTO.login(), "login should have length");
        Assert.hasLength(userAccountDTO.email(), "email should have length");
    }

    public record UpdateUserAccountEntityNotEmptyResponse(
        UserAccount userAccount,
        boolean wasEmailSet
    ){}

    private static String trimToNull(String input) {
        if (input == null) {
            return null;
        }
        var ret = input.trim();
        if (ret.isEmpty()) {
            return null;
        }
        return ret;
    }

    // EditUserDTO userAccountDTO is already filtered through normalize()
    public static UpdateUserAccountEntityNotEmptyResponse updateUserAccountEntityNotEmpty(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        var wasEmailSet = false;
        if (StringUtils.hasLength(userAccountDTO.login())) {
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
            if (!userAccountDTO.email().equals(userAccount.email())) {
                userAccount = userAccount.withNewEmail(userAccountDTO.email());
                wasEmailSet = true;
            } else {
                userAccount = userAccount.withNewEmail(null);
            }
        }
        if (StringUtils.hasLength(userAccountDTO.shortInfo())) {
            userAccount = userAccount.withShortInfo(userAccountDTO.shortInfo());
        }
        if (Boolean.TRUE.equals(userAccountDTO.removeLoginColor())) {
            userAccount = userAccount.withLoginColor(null);
        } else if (StringUtils.hasLength(userAccountDTO.loginColor())) {
            userAccount = userAccount.withLoginColor(userAccountDTO.loginColor());
        }

        return new UpdateUserAccountEntityNotEmptyResponse(userAccount, wasEmailSet);
    }

    public static com.github.nkonev.aaa.dto.EditUserDTO convertToEditUserDto(UserAccount userAccount) {
        return new com.github.nkonev.aaa.dto.EditUserDTO(
                userAccount.username(),
                userAccount.avatar(),
                null,
                null,
                userAccount.email(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                userAccount.loginColor(),
                null
        );
    }

}
