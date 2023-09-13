import mitt from 'mitt'

const emitter = mitt()

export default emitter


export const LOGGED_OUT = "loggedOut";
export const PROFILE_SET = "profileSetNotNull";
export const LOGGED_IN = "loggedIn";

export const SEARCH_STRING_CHANGED = "searchStringChanged";

export const OPEN_NOTIFICATIONS_DIALOG = "openNotifications";

export const OPEN_TEXT_EDIT_MODAL = "openTextEditModal";

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
export const OPEN_EDIT_MESSAGE = "openEditMessage";
export const CLOSE_EDIT_MESSAGE = "closeEditMessage";
export const SET_FILE_ITEM_UUID = "setFileItemUuid";
export const INCREMENT_FILE_ITEM_FILE_COUNT = "incrementFileItemFileCount";
export const SET_FILE_ITEM_FILE_COUNT = "setFileItemFileCount";
export const MEDIA_LINK_SET = "mediaLinkSet";
export const EMBED_LINK_SET = "embedLinkSet";
export const PREVIEW_CREATED = "previewCreated";
export const FILE_UPLOAD_MODAL_START_UPLOADING = "FileUploadModalStartUpload";

export const OPEN_SETTINGS = "openSettings";

export const REQUEST_CHANGE_VIDEO_PARAMETERS = "requestChangeVideoParameters";
export const VIDEO_PARAMETERS_CHANGED = "videoParametersChanged";

export const OPEN_SIMPLE_MODAL = "openSimpleModal";
export const CLOSE_SIMPLE_MODAL = "closeSimpleModal";
export const VIDEO_CALL_USER_COUNT_CHANGED = "videoCallUserCountChanged";
export const VIDEO_CALL_SCREEN_SHARE_CHANGED = "videoCallScreenShareChanged";

export const VIDEO_RECORDING_CHANGED = "videoRecordingChanged";
export const VIDEO_CALL_INVITED = "videoCallInvited";
export const VIDEO_DIAL_STATUS_CHANGED = "videoDialStatusChanged";
export const UNREAD_MESSAGES_CHANGED = "unreadMessagesChanged";
export const PLAYER_MODAL = "playerModal";
export const OPEN_PARTICIPANTS_DIALOG = "openInfo";


export const NOTIFICATION_ADD = "notificationAdd";
export const NOTIFICATION_DELETE = "notificationDelete";


export const WEBSOCKET_RESTORED = "wsRestored";

export const CHAT_ADD = "chatAdd";
export const CHAT_EDITED = "chatEdited";
export const CHAT_DELETED = "chatDeleted";

export const MESSAGE_ADD = "messageAdd";
export const MESSAGE_DELETED = "messageDeleted";
export const MESSAGE_EDITED = "messageEdited";
export const USER_TYPING = "userTyping";
export const MESSAGE_BROADCAST = "messageBroadcast";
export const PARTICIPANT_ADDED = "participantAdded";
export const PARTICIPANT_DELETED = "participantDeleted";
export const PARTICIPANT_EDITED = "participantEdited";
export const PINNED_MESSAGE_PROMOTED = "pinnedMessagePromoted";
export const PINNED_MESSAGE_UNPROMOTED = "pinnedMessageUnpromoted";
export const FILE_CREATED = "fileCreated";
export const FILE_REMOVED = "fileRemoved";

export const ATTACH_FILES_TO_MESSAGE_MODAL = "attachFilesToMessage";
export const OPEN_CHAT_EDIT = "openChatEdit";
export const OPEN_FIND_USER = "openFindUser";

export const LOAD_FILES_COUNT = "loadFilesCount";

export const OPEN_MESSAGE_READ_USERS_DIALOG = "openMessageReadUsersDialog";
