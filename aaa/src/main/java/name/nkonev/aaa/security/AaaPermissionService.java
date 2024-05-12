package name.nkonev.aaa.security;

import name.nkonev.aaa.Constants;
import name.nkonev.aaa.dto.ConfirmDTO;
import name.nkonev.aaa.repository.jdbc.UserAccountRepository;
import name.nkonev.aaa.dto.LockDTO;
import name.nkonev.aaa.dto.UserAccountDetailsDTO;
import name.nkonev.aaa.entity.jdbc.UserAccount;
import jakarta.annotation.PostConstruct;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.Optional;

/**
 * Central entrypoint for access decisions
 */
@Service
public class AaaPermissionService {

    @Autowired
    private UserRoleService userRoleService;

    @Autowired
    private UserAccountRepository userAccountRepository;

    private UserAccount deleted;

    @PostConstruct
    public void postConstruct() {
        deleted = userAccountRepository.findByUsername(Constants.DELETED).orElseThrow();
    }

    public boolean hasSessionManagementPermission(PrincipalToCheck userAccount) {
        if (userAccount==null){
            return false;
        }
        return userAccount.isAdmin();
    }

    public boolean hasSessionManagementPermission(UserAccountDetailsDTO userAccount) {
        return hasSessionManagementPermission(PrincipalToCheck.ofUserAccount(userAccount, userRoleService));
    }

    public boolean canLock(UserAccountDetailsDTO userAccount, LockDTO lockDTO) {
        if (userAccount==null){
            return false;
        }
        if (lockDTO==null){
            return false;
        }
        return lockAndDelete(PrincipalToCheck.ofUserAccount(userAccount, userRoleService), lockDTO.userId());
    }

    public boolean canConfirm(UserAccountDetailsDTO userAccount, ConfirmDTO confirmDTO) {
        if (userAccount==null){
            return false;
        }
        if (confirmDTO==null){
            return false;
        }
        return lockAndDelete(PrincipalToCheck.ofUserAccount(userAccount, userRoleService), confirmDTO.userId());
    }

    public boolean canDelete(UserAccountDetailsDTO userAccount, long userIdToDelete) {
        return lockAndDelete(PrincipalToCheck.ofUserAccount(userAccount, userRoleService), userIdToDelete);
    }

    public boolean canSelfDelete(UserAccountDetailsDTO userAccount) {
        return Optional.ofNullable(userAccount).isPresent();
    }

    public boolean canChangeRole(UserAccountDetailsDTO currentUser, long userAccountId) {
        return lockAndDelete(PrincipalToCheck.ofUserAccount(currentUser, userRoleService), userAccountId);
    }

    public boolean canLock(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        return lockAndDelete(currentUser, userAccount.id());
    }

    public boolean canConfirm(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        return lockAndDelete(currentUser, userAccount.id());
    }

    public boolean canDelete(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        return lockAndDelete(currentUser, userAccount.id());
    }

    public boolean canChangeRole(PrincipalToCheck currentUser, UserAccount userAccount) {
        if (userAccount == null) {
            return false;
        }
        return lockAndDelete(currentUser, userAccount.id());
    }

    private boolean lockAndDelete(PrincipalToCheck currentUser, Long subjectUserAccountId) {
        var maybeUserAccount = userAccountRepository.findById(subjectUserAccountId);
        if (maybeUserAccount.isEmpty()) {
            return false;
        }

        if (subjectUserAccountId == null) {
            return false;
        }
        if (currentUser == null) {
            return false;
        }
        if (deleted.id().equals(subjectUserAccountId)){
            return false;
        }
        if (subjectUserAccountId.equals(currentUser.getId())){
            return false;
        }
        return currentUser.isAdmin();
    }
}
