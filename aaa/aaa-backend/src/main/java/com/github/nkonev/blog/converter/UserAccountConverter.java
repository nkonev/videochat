package com.github.nkonev.blog.converter;

import com.github.nkonev.blog.ApiConstants;
import com.github.nkonev.blog.dto.*;
import com.github.nkonev.blog.entity.jdbc.CreationType;
import com.github.nkonev.blog.entity.jdbc.UserAccount;
import com.github.nkonev.blog.dto.UserRole;
import com.github.nkonev.blog.exception.BadRequestException;
import com.github.nkonev.blog.repository.jdbc.UserAccountRepository;
import com.github.nkonev.blog.security.BlogSecurityService;
import com.github.nkonev.blog.security.FacebookOAuth2UserService;
import com.github.nkonev.blog.security.VkontakteOAuth2UserService;
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
    private BlogSecurityService blogSecurityService;

    @Autowired
    private UserAccountRepository userAccountRepository;

    private static UserRole getDefaultUserRole(){
        return UserRole.ROLE_USER;
    }

    private static OauthIdentifiersDTO convertOauth(UserAccount.OauthIdentifiers oauthIdentifiers){
        if (oauthIdentifiers==null) return null;
        return new OauthIdentifiersDTO(oauthIdentifiers.getFacebookId(), oauthIdentifiers.getVkontakteId());
    }

    private static UserAccount.OauthIdentifiers convertOauth(OauthIdentifiersDTO oauthIdentifiers){
        if (oauthIdentifiers==null) return null;
        return new UserAccount.OauthIdentifiers(oauthIdentifiers.getFacebookId(), oauthIdentifiers.getVkontakteId());
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
                convertOauth(userAccount.getOauthIdentifiers())
        );
    }

    public static UserSelfProfileDTO getUserSelfProfile(UserAccountDetailsDTO userAccount, LocalDateTime lastLoginDateTime, Long expiresAt) {
        if (userAccount == null) { return null; }
        return new UserSelfProfileDTO (
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getEmail(),
                lastLoginDateTime,
                userAccount.getOauthIdentifiers(),
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

    public UserAccountDTO convertToUserAccountDTO(UserAccount userAccount) {
        if (userAccount == null) { return null; }
        return new UserAccountDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getLastLoginDateTime(),
                convertOauth(userAccount.getOauthIdentifiers())
        );
    }

    public OwnerDTO convertToOwnerDTO(Long ownerId) {
        if (ownerId == null) { return null; }
        Optional<UserAccount> byId = userAccountRepository.findById(ownerId);
        return byId.map(userAccount -> new OwnerDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar()
        )).orElse(null);
    }

    public UserAccountDTOExtended convertToUserAccountDTOExtended(UserAccountDetailsDTO currentUser, UserAccount userAccount) {
        if (userAccount == null) { return null; }
        UserAccountDTOExtended.DataDTO dataDTO;
        if (blogSecurityService.hasSessionManagementPermission(currentUser)){
            dataDTO = new UserAccountDTOExtended.DataDTO(userAccount.isEnabled(), userAccount.isExpired(), userAccount.isLocked(), userAccount.getRole());
        } else {
            dataDTO = null;
        }
        return new UserAccountDTOExtended(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                dataDTO,
                userAccount.getLastLoginDateTime(),
                convertOauth(userAccount.getOauthIdentifiers()),
                blogSecurityService.canLock(currentUser, userAccount),
                blogSecurityService.canDelete(currentUser, userAccount),
                blogSecurityService.canChangeRole(currentUser, userAccount)
        );
    }

    public static UserAccountDTO convertToUserAccountDTO(UserAccountDetailsDTO userAccount) {
        if (userAccount == null) { return null; }
        return new UserAccountDTO(
                userAccount.getId(),
                userAccount.getUsername(),
                userAccount.getAvatar(),
                userAccount.getLastLoginDateTime(),
                userAccount.getOauthIdentifiers()
        );
    }


    private static void validateUserPassword(String password) {
        Assert.notNull(password, "password must be set");
        if (password.length() < ApiConstants.MIN_PASSWORD_LENGTH || password.length() > ApiConstants.MAX_PASSWORD_LENGTH) {
            throw new BadRequestException("password don't match requirements");
        }
    }

    public static UserAccount buildUserAccountEntityForInsert(EditUserDTO userAccountDTO, PasswordEncoder passwordEncoder) {
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


    private static void validateAndTrimLogin(EditUserDTO userAccountDTO) {
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
                new UserAccount.OauthIdentifiers(facebookId, null)
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
                new UserAccount.OauthIdentifiers(null, vkontakteId)
        );
    }


    private static void validateLoginAndEmail(EditUserDTO userAccountDTO){
        Assert.hasLength(userAccountDTO.getLogin(), "login should have length");
        Assert.hasLength(userAccountDTO.getEmail(), "email should have length");
    }

    public static void updateUserAccountEntity(EditUserDTO userAccountDTO, UserAccount userAccount, PasswordEncoder passwordEncoder) {
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

    public static EditUserDTO convertToEditUserDto(UserAccount userAccount) {
        EditUserDTO e = new EditUserDTO();
        e.setAvatar(userAccount.getAvatar());
        e.setEmail(userAccount.getEmail());
        e.setLogin(userAccount.getUsername());
        return e;
    }

}
