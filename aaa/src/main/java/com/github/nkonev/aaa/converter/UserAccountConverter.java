package com.github.nkonev.aaa.converter;

import com.github.nkonev.aaa.Constants;
import com.github.nkonev.aaa.dto.OAuth2IdentifiersDTO;
import com.github.nkonev.aaa.dto.UserAccountDTOExtended;
import com.github.nkonev.aaa.dto.UserAccountDetailsDTO;
import com.github.nkonev.aaa.entity.jdbc.CreationType;
import com.github.nkonev.aaa.entity.jdbc.UserAccount;
import com.github.nkonev.aaa.dto.UserRole;
import com.github.nkonev.aaa.exception.BadRequestException;
import com.github.nkonev.aaa.repository.jdbc.UserAccountRepository;
import com.github.nkonev.aaa.security.AaaSecurityService;
import com.github.nkonev.aaa.security.FacebookOAuth2UserService;
import com.github.nkonev.aaa.security.VkontakteOAuth2UserService;
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

    @Autowired
    private UserAccountRepository userAccountRepository;

    private static UserRole getDefaultUserRole(){
        return UserRole.ROLE_USER;
    }

    public static List<String> convertRolesToStringList(Collection<GrantedAuthority> roles) {
        return Optional.ofNullable(roles).map(rs -> rs.stream().map(GrantedAuthority::getAuthority).collect(Collectors.toList())).orElse(Collections.emptyList());
    }

    private static OAuth2IdentifiersDTO convertOauth(UserAccount.OAuth2Identifiers oAuth2Identifiers){
        if (oAuth2Identifiers ==null) return null;
        return new OAuth2IdentifiersDTO(oAuth2Identifiers.getFacebookId(), oAuth2Identifiers.getVkontakteId(), oAuth2Identifiers.getGoogleId());
    }

    public static UserAccountDetailsDTO convertToUserAccountDetailsDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        return new UserAccountDetailsDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getPassword(),
                userAccount.isExpired(),
                userAccount.isLocked(),
                userAccount.isEnabled(),
                Collections.singletonList(convertRole(userAccount.getRole())),
                userAccount.getEmail(),
                userAccount.getLastLoginDateTime(),
                convertOauth(userAccount.getOauth2Identifiers())
        );
    }

    public static com.github.nkonev.aaa.dto.UserSelfProfileDTO getUserSelfProfile(UserAccountDetailsDTO userAccount, LocalDateTime lastLoginDateTime, Long expiresAt) {
        if (userAccount == null) { return null; }
        return new com.github.nkonev.aaa.dto.UserSelfProfileDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
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
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getLastLoginDateTime(),
                convertOauth(userAccount.getOauth2Identifiers())
        );
    }

    public com.github.nkonev.aaa.dto.OwnerDTO convertToOwnerDTO(Long ownerId) {
        if (ownerId == null) { return null; }
        Optional<UserAccount> byId = userAccountRepository.findById(ownerId);
        return byId.map(userAccount -> new com.github.nkonev.aaa.dto.OwnerDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar()
        )).orElse(null);
    }

    public com.github.nkonev.aaa.dto.UserAccountDTOExtended convertToUserAccountDTOExtended(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO dataDTO;
        if (aaaSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new com.github.nkonev.aaa.dto.UserAccountDTOExtended.DataDTO(userAccount.isEnabled(), userAccount.isExpired(), userAccount.isLocked(), userAccount.getRole());
        } else {
            dataDTO = null;
        }
        return new UserAccountDTOExtended(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                dataDTO,
                userAccount.getLastLoginDateTime(),
                convertOauth(userAccount.getOauth2Identifiers()),
                aaaSecurityService.canLock(currentUser, userAccount),
                aaaSecurityService.canDelete(currentUser, userAccount),
                aaaSecurityService.canChangeRole(currentUser, userAccount)
        );
    }

    public static com.github.nkonev.aaa.dto.UserAccountDTO convertToUserAccountDTO(UserAccountDetailsDTO userAccount) {
        if (userAccount == null) { return null; }
        return new com.github.nkonev.aaa.dto.UserAccountDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getLastLoginDateTime(),
                userAccount.getOauth2Identifiers()
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
        validateAndTrimLogin(userAccountDTO);
        String password = userAccountDTO.getPassword();
        try {
            validateUserPassword(password);
        } catch (IllegalArgumentException e) {
            throw new BadRequestException(e.getMessage());
        }

        return new UserAccount(
                CreationType.REGISTRATION,
                userAccountDTO.getLogin(),
                passwordEncoder.encode(password),
                userAccountDTO.getAvatar(),
                expired,
                locked,
                enabled,
                newUserRole,
                userAccountDTO.getEmail(),
                null
        );
    }

    public static String validateAndTrimLogin(String login){
        Assert.notNull(login, "login cannot be null");
        login = login.trim();
        Assert.hasLength(login, "login should have length");
        Assert.isTrue(!login.startsWith(FacebookOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");
        Assert.isTrue(!login.startsWith(VkontakteOAuth2UserService.LOGIN_PREFIX), "not allowed prefix");

        return login;
    }


    private static void validateAndTrimLogin(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO) {
        userAccountDTO.setLogin(validateAndTrimLogin(userAccountDTO.getLogin()));
    }

    // used for just get user id
    public static UserAccount buildUserAccountEntityForFacebookInsert(String facebookId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                CreationType.FACEBOOK,
                login,
                null,
                maybeImageUrl,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                new UserAccount.OAuth2Identifiers(facebookId, null, null)
        );
    }

    public static UserAccount buildUserAccountEntityForVkontakteInsert(String vkontakteId, String login) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                CreationType.VKONTAKTE,
                login,
                null,
                null,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                new UserAccount.OAuth2Identifiers(null, vkontakteId, null)
        );
    }

    public static UserAccount buildUserAccountEntityForGoogleInsert(String googleId, String login, String maybeImageUrl) {
        final boolean expired = false;
        final boolean locked = false;
        final boolean enabled = true;

        final UserRole newUserRole = getDefaultUserRole();

        return new UserAccount(
                CreationType.GOOGLE,
                login,
                null,
                maybeImageUrl,
                expired,
                locked,
                enabled,
                newUserRole,
                null,
                new UserAccount.OAuth2Identifiers(null, null, googleId)
        );
    }

    private static void validateLoginAndEmail(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO){
        Assert.hasLength(userAccountDTO.getLogin(), "login should have length");
        Assert.hasLength(userAccountDTO.getEmail(), "email should have length");
    }

    public static void updateUserAccountEntity(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        Assert.hasLength(userAccountDTO.getLogin(), "login should have length");
        validateAndTrimLogin(userAccountDTO);
        String password = userAccountDTO.getPassword();
        if (!StringUtils.isEmpty(password)) {
            validateUserPassword(password);
            userAccount.setPassword(passwordEncoder.encode(password));
        }
        userAccount.setUsername(userAccountDTO.getLogin());
        if (Boolean.TRUE.equals(userAccountDTO.getRemoveAvatar())){
            userAccount.setAvatar(null);
        } else {
            userAccount.setAvatar(userAccountDTO.getAvatar());
        }
        if (!StringUtils.isEmpty(userAccountDTO.getEmail())) {
            String email = userAccountDTO.getEmail();
            email = email.trim();
            userAccount.setEmail(email);
        }
    }

    public static void updateUserAccountEntityNotEmpty(com.github.nkonev.aaa.dto.EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
        if (!StringUtils.isEmpty(userAccountDTO.getLogin())) {
            validateAndTrimLogin(userAccountDTO);
            userAccount.setUsername(userAccountDTO.getLogin());
        }
        String password = userAccountDTO.getPassword();
        if (!StringUtils.isEmpty(password)) {
            validateUserPassword(password);
            userAccount.setPassword(passwordEncoder.encode(password));
        }
        if (Boolean.TRUE.equals(userAccountDTO.getRemoveAvatar())){
            userAccount.setAvatar(null);
        } else if (!StringUtils.isEmpty(userAccountDTO.getAvatar())) {
            userAccount.setAvatar(userAccountDTO.getAvatar());
        }
        if (!StringUtils.isEmpty(userAccountDTO.getEmail())) {
            String email = userAccountDTO.getEmail();
            email = email.trim();
            userAccount.setEmail(email);
        }
    }

    public static com.github.nkonev.aaa.dto.EditUserDTO convertToEditUserDto(UserAccount userAccount) {
        com.github.nkonev.aaa.dto.EditUserDTO e = new com.github.nkonev.aaa.dto.EditUserDTO();
        e.setAvatar(userAccount.getAvatar());
        e.setEmail(userAccount.getEmail());
        e.setLogin(userAccount.getUsername());
        return e;
    }

}
