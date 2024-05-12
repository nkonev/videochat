package name.nkonev.aaa.security;

import name.nkonev.aaa.dto.UserAccountDetailsDTO;

import static name.nkonev.aaa.Constants.NonExistentUser;

public sealed interface PrincipalToCheck permits KnownAdmin, UserToCheck {
    boolean isAdmin();

    Long getId();

    static PrincipalToCheck knownAdmin() {
        return new KnownAdmin();
    }

    static PrincipalToCheck ofUserAccount(UserAccountDetailsDTO userAccount, UserRoleService userRoleService) {
        return new UserToCheck(userAccount, userRoleService);
    }
}

final class KnownAdmin implements PrincipalToCheck {

    @Override
    public boolean isAdmin() {
        return true;
    }

    @Override
    public Long getId() {
        return NonExistentUser;
    }
}

final class UserToCheck implements PrincipalToCheck {

    private final UserAccountDetailsDTO userAccount;

    private final UserRoleService userRoleService;

    UserToCheck(UserAccountDetailsDTO userAccount, UserRoleService userRoleService) {
        this.userAccount = userAccount;
        this.userRoleService = userRoleService;
    }

    @Override
    public boolean isAdmin() {
        if (userAccount == null) {
            return false;
        }
        return userRoleService.isAdmin(userAccount);
    }

    @Override
    public Long getId() {
        if (userAccount == null) {
            return null;
        }
        return userAccount.getId();
    }
}
