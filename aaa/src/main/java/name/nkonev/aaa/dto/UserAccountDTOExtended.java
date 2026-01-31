package name.nkonev.aaa.dto;

import com.fasterxml.jackson.annotation.JsonUnwrapped;

public record UserAccountDTOExtended (
    @JsonUnwrapped
    UserAccountDTO userAccountDTO,

    // can I as an user / admin lock him ?
    boolean canLock,
    boolean canEnable,

    boolean canDelete,

    boolean canChangeRole,
    boolean canConfirm,
    boolean awaitingForConfirmEmailChange,
    boolean canRemoveSessions,
    boolean canSetPassword, // set somebody's password

    boolean canChangeSelfLogin,
    boolean canChangeSelfEmail,
    boolean canChangeSelfPassword,

    boolean canChangeOverriddenPermissions
) {

}
