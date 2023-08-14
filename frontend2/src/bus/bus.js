import mitt from 'mitt'

const emitter = mitt()

export default emitter


export const LOGGED_OUT = "loggedOut";
export const PROFILE_SET = "profileSetNotNull";
export const LOGGED_IN = "loggedIn";

export const SEARCH_STRING_CHANGED = "searchStringChanged";

export const OPEN_NOTIFICATIONS_DIALOG = "openNotifications";


export const SCROLL_DOWN = "scrollDown";

export const OPEN_MESSAGE_EDIT_LINK = "openMessageEditLink";
export const MESSAGE_EDIT_LINK_SET = "messageEditLinkSet";
export const OPEN_MESSAGE_EDIT_COLOR = "messageEditColor";
export const OPEN_MESSAGE_EDIT_SMILEY = "messageEditSmiley";
export const MESSAGE_EDIT_COLOR_SET = "messageEditColorSet";
export const OPEN_FILE_UPLOAD_MODAL = "openFileUploadModal";
export const CLOSE_FILE_UPLOAD_MODAL = "closeFileUploadModal";
export const OPEN_MESSAGE_EDIT_MEDIA = "openMessageEditMedia";
export const OPEN_VIEW_FILES_DIALOG = "openViewFiles";
export const SET_EDIT_MESSAGE = "setEditMessageDto";
export const CLOSE_EDIT_MESSAGE = "closeEditMessage";
export const SET_FILE_ITEM_UUID = "setFileItemUuid";
