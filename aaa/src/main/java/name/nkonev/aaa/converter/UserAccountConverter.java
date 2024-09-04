package name.nkonev.aaa.converter;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.config.properties.AaaProperties;
import name.nkonev.aaa.dto.*;
import name.nkonev.aaa.entity.jdbc.CreationType;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import name.nkonev.aaa.exception.BadRequestException;
import name.nkonev.aaa.repository.redis.ChangeEmailConfirmationTokenRepository;
import name.nkonev.aaa.security.*;
import name.nkonev.aaa.utils.*;
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

import static name.nkonev.aaa.Constants.FORBIDDEN_USERNAMES;
import static name.nkonev.aaa.Constants.FORBIDDEN_USERNAME_PREFIXES;
import static name.nkonev.aaa.utils.NullUtils.trimToNull;
import static name.nkonev.aaa.utils.RoleUtils.DEFAULT_ROLE;
import static name.nkonev.aaa.security.AaaPermissionService.canAccessToAdminsCorner;

@Component
public class UserAccountConverter {

    @Autowired
    private AaaPermissionService aaaSecurityService;

    @Autowired
    private AaaProperties aaaProperties;

    @Autowired
    private ChangeEmailConfirmationTokenRepository changeEmailConfirmationTokenRepository;

    static final UserRole[] newUserRoles = new UserRole[]{DEFAULT_ROLE};

    public static String normalizeEmail(String email) {
        return trimToNull(NullEncode.forHtmlEmail(email));
    }

    public static String normalizeLogin(String login) {
        login = login != null ? login.trim() : null;
        login = NullEncode.forHtmlLogin(login);
        login = trimToNull(login);
        return login;
    }

    public static EditUserDTO normalize(EditUserDTO editUserDTO, boolean isExternalIntegration) {
        var userAccountDTO = editUserDTO.withLogin(checkAndTrimLogin(editUserDTO.login(), isExternalIntegration));
        userAccountDTO = userAccountDTO.withEmail(normalizeEmail(userAccountDTO.email()));
        userAccountDTO = userAccountDTO.withAvatar(trimToNull(NullEncode.forHtmlAttribute(userAccountDTO.avatar())));
        userAccountDTO = userAccountDTO.withAvatarBig(trimToNull(NullEncode.forHtmlAttribute(userAccountDTO.avatarBig())));
        userAccountDTO = userAccountDTO.withShortInfo(trimToNull(NullEncode.forHtml(userAccountDTO.shortInfo())));
        userAccountDTO = userAccountDTO.withLoginColor(trimToNull(NullEncode.forHtml(userAccountDTO.loginColor())));
        return userAccountDTO;
    }

    public static List<String> convertRolesToStringList(Collection<GrantedAuthority> roles) {
        return Optional.ofNullable(roles).map(rs -> rs.stream().map(GrantedAuthority::getAuthority).collect(Collectors.toList())).orElse(Collections.emptyList());
    }

    private static OAuth2IdentifiersDTO convertOAuth2(UserAccount.OAuth2Identifiers oAuth2Identifiers){
        if (oAuth2Identifiers ==null) return null;
        return new OAuth2IdentifiersDTO(oAuth2Identifiers.facebookId(), oAuth2Identifiers.vkontakteId(), oAuth2Identifiers.googleId(), oAuth2Identifiers.keycloakId());
    }

    private boolean awaitingForConfirmEmailChange(long userId) {
        return changeEmailConfirmationTokenRepository.findById(userId).map(t -> StringUtils.hasLength(t.newEmail())).orElse(false);
    }

    public UserAccountDetailsDTO convertToUserAccountDetailsDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        var awaitingForConfirmEmailChange = awaitingForConfirmEmailChange(userAccount.id());
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
                Arrays.stream(userAccount.roles()).map(UserAccountConverter::convertRole).collect(Collectors.toSet()),
                userAccount.email(),
                awaitingForConfirmEmailChange,
                userAccount.lastLoginDateTime(),
                convertOAuth2(userAccount.oauth2Identifiers()),
                userAccount.loginColor(),
                userAccount.ldapId()
        );
    }

    public static name.nkonev.aaa.dto.UserSelfProfileDTO getUserSelfProfile(UserAccountDetailsDTO userAccount, LocalDateTime lastLoginDateTime, Long expiresAt) {
        if (userAccount == null) { return null; }
        var roles = convertRoles2Enum(userAccount.getRoles());
        var canShowAdminsCorner = canAccessToAdminsCorner(roles);
        return new name.nkonev.aaa.dto.UserSelfProfileDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getAvatarBig(),
                userAccount.userAccountDTO().shortInfo(),
                userAccount.getEmail(),
                userAccount.awaitingForConfirmEmailChange(),
                lastLoginDateTime,
                userAccount.getOauth2Identifiers(),
                roles,
                expiresAt,
                userAccount.getLoginColor(),
                userAccount.ldapId() != null,
                canShowAdminsCorner
        );
    }

    public static Collection<UserRole> convertRoles2Enum(Collection<GrantedAuthority> roles) {
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

    public static name.nkonev.aaa.dto.UserAccountDTO convertToUserAccountDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        return new name.nkonev.aaa.dto.UserAccountDTO(
                userAccount.id(),
                userAccount.username(),
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                userAccount.lastLoginDateTime(),
                convertOAuth2(userAccount.oauth2Identifiers()),
                userAccount.loginColor(),
                userAccount.ldapId() != null
        );
    }

    public name.nkonev.aaa.dto.UserAccountEventDTO convertToUserAccountEventDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        var awaitingForConfirmEmailChange = awaitingForConfirmEmailChange(userAccount.id());
        return new name.nkonev.aaa.dto.UserAccountEventDTO(
                userAccount.id(),
                userAccount.username(),
                userAccount.email(),
                awaitingForConfirmEmailChange,
                userAccount.avatar(),
                userAccount.avatarBig(),
                userAccount.shortInfo(),
                userAccount.lastLoginDateTime(),
                convertOAuth2(userAccount.oauth2Identifiers()),
                userAccount.loginColor(),
                userAccount.ldapId() != null
        );
    }

    public name.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtended(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        name.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new name.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.enabled(), userAccount.expired(), userAccount.locked(), userAccount.confirmed(), Arrays.stream(userAccount.roles()).collect(Collectors.toSet()));
        } else {
            dataDTO = null;
        }
        var awaitingForConfirmEmailChange = awaitingForConfirmEmailChange(userAccount.id());
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
                aaaSecurityService.canEnable(currentUser, userAccount),
                aaaSecurityService.canDelete(currentUser, userAccount),
                aaaSecurityService.canChangeRole(currentUser, userAccount),
                aaaSecurityService.canConfirm(currentUser, userAccount),
                awaitingForConfirmEmailChange,
                userAccount.loginColor(),
                aaaSecurityService.canRemoveSessions(currentUser, userAccount.id()),
                userAccount.ldapId() != null
        );
    }

    private static void validateUserPassword(String password) {
        Assert.notNull(password, "password must be set");
        if (password.length() < Constants.MIN_PASSWORD_LENGTH || password.length() > Constants.MAX_PASSWORD_LENGTH) {
            throw new BadRequestException("password don't match requirements");
        }
    }

    // EditUserDTO userAccountDTO is already filtered by normalize()
    public static UserAccount buildUserAccountEntityForInsert(name.nkonev.aaa.dto.EditUserDTO userAccountDTO, PasswordEncoder passwordEncoder) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = false;

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
                newUserRoles,
                userAccountDTO.email(),
                null,
                null,
                null,
                null,
                null,
                null,
                userAccountDTO.loginColor(),
                null,
                null,
                null,
                null
        );
    }

    public static String validateLengthAndTrimLogin(String login, boolean isExternalIntegration) {
        login = checkAndTrimLogin(login, isExternalIntegration);

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

    private static String checkAndTrimLogin(String login, boolean isExternalIntegration) {
        login = normalizeLogin(login);

        if (login != null) {
            if (FORBIDDEN_USERNAMES.contains(login)) {
                throw new BadRequestException("forbidden login");
            }

            for (var c: login.chars().toArray()) {
                if (
                    !Character.isLetterOrDigit(c) &&
                    c != '_' &&
                    c != ' ' &&
                    c != '-' &&
                    c != '+' &&
                    c != '/' &&
                    c != '*' &&
                    c != '=' &&
                    c != '!' &&
                    c != '?' &&
                    c != '$' &&
                    c != '^' &&
                    c != '&' &&
                    c != '@' &&
                    c != '.' &&
                    c != ',' &&
                    c != '#'
                ) {
                    throw new BadRequestException("login contains invalid character");
                }
            }
        }

        if (login != null && !isExternalIntegration) {
            for (var fp : FORBIDDEN_USERNAME_PREFIXES) {
                if (login.startsWith(fp)) {
                    throw new BadRequestException("not allowed prefix");
                }
            }
        }

        return login;
    }

    // used for just get user id
    public static UserAccount buildUserAccountEntityForFacebookInsert(String facebookId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;
        final boolean confirmed = true;

        return new UserAccount(
                null,
                CreationType.FACEBOOK,
                normalizeLogin(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRoles,
                null,
                null,
                facebookId,
                null,
                null,
                null,
                null,
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

        return new UserAccount(
                null,
                CreationType.VKONTAKTE,
                normalizeLogin(login),
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRoles,
                null,
                null,
                null,
                vkontakteId,
                null,
                null,
                null,
                null,
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

        return new UserAccount(
                null,
                CreationType.GOOGLE,
                normalizeLogin(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                newUserRoles,
                null,
                null,
                null,
                null,
                googleId,
                null,
                null,
                null,
                null,
                null,
                null,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForKeycloakInsert(String keycloakId, String login, String maybeImageUrl, Set<UserRole> roles, String email, boolean locked, boolean enabled, LocalDateTime syncKeycloakTime) {
        final boolean expired = false;
        final boolean confirmed = true;

        return new UserAccount(
                null,
                CreationType.KEYCLOAK,
                normalizeLogin(login),
                null,
                NullEncode.forHtmlAttribute(maybeImageUrl),
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                roles.toArray(UserRole[]::new),
                normalizeEmail(email),
                null,
                null,
                null,
                null,
                keycloakId,
                null,
                null,
                null,
                syncKeycloakTime,
                syncKeycloakTime,
                null
        );
    }

    public static UserAccount buildUserAccountEntityForLdapInsert(String login, String ldapId, Set<UserRole> mappedRoles, String email, boolean locked, boolean enabled, LocalDateTime syncLdapTime) {
        final boolean expired = false;
        final boolean confirmed = true;

        return new UserAccount(
                null,
                CreationType.LDAP,
                normalizeLogin(login),
                null,
                null,
                null,
                null,
                expired,
                locked,
                enabled,
                confirmed,
                mappedRoles.toArray(UserRole[]::new),
                normalizeEmail(email),
                null,
                null,
                null,
                null,
                null,
                ldapId,
                null,
                syncLdapTime,
                null,
                null,
                syncLdapTime
        );
    }

    private static void validateLoginAndEmail(name.nkonev.aaa.dto.EditUserDTO userAccountDTO){
        Assert.hasLength(userAccountDTO.login(), "login should have length");
        Assert.hasLength(userAccountDTO.email(), "email should have length");
    }

    public record UpdateUserAccountEntityNotEmptyResponse(
        UserAccount userAccount,
        String newEmail,
        Action action
    ){
        public enum Action {
            NO_ACTION,
            NEW_EMAIL_WAS_SET,
            SHOULD_REMOVE_NEW_EMAIL
        }
    }

    // EditUserDTO userAccountDTO is already filtered through normalize()
    public UpdateUserAccountEntityNotEmptyResponse updateUserAccountEntityNotEmpty(name.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        var emailAction = UpdateUserAccountEntityNotEmptyResponse.Action.NO_ACTION;
        String newEmail = null;
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
            userAccount = userAccount.withAvatar(filterAvatar(userAccountDTO.avatar()));
            userAccount = userAccount.withAvatarBig(filterAvatar(userAccountDTO.avatarBig()));
        }
        if (StringUtils.hasLength(userAccountDTO.email())) {
            if (!userAccountDTO.email().equals(userAccount.email())) {
                newEmail = userAccountDTO.email();
                emailAction = UpdateUserAccountEntityNotEmptyResponse.Action.NEW_EMAIL_WAS_SET;
            } else {
                emailAction = UpdateUserAccountEntityNotEmptyResponse.Action.SHOULD_REMOVE_NEW_EMAIL;
            }
        }
        if (Boolean.TRUE.equals(userAccountDTO.removeShortInfo())){
            userAccount = userAccount.withShortInfo(null);
        } else if (StringUtils.hasLength(userAccountDTO.shortInfo())) {
            userAccount = userAccount.withShortInfo(userAccountDTO.shortInfo());
        }
        if (Boolean.TRUE.equals(userAccountDTO.removeLoginColor())) {
            userAccount = userAccount.withLoginColor(null);
        } else if (StringUtils.hasLength(userAccountDTO.loginColor())) {
            userAccount = userAccount.withLoginColor(userAccountDTO.loginColor());
        }

        return new UpdateUserAccountEntityNotEmptyResponse(userAccount, newEmail, emailAction);
    }

    private String filterAvatar(String input) {
        var allowedUrls = aaaProperties.getAllowedAvatarUrlsList();
        if (UrlUtils.containsUrl(allowedUrls, input)) {
            return input;
        } else {
            return null;
        }
    }

    public static name.nkonev.aaa.dto.EditUserDTO convertToEditUserDto(UserAccount userAccount) {
        return new name.nkonev.aaa.dto.EditUserDTO(
                userAccount.username(),
                userAccount.avatar(),
                null,
                null,
                userAccount.email(),
                userAccount.avatarBig(),
                null,
                userAccount.shortInfo(),
                userAccount.loginColor(),
                null
        );
    }

}
