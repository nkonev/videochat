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
export const OPEN_CHOOSE_COLOR = "chooseColor";
export const OPEN_MESSAGE_EDIT_SMILEY = "messageEditSmiley";
export const COLOR_SET = "messageEditColorSet";
export const OPEN_FILE_UPLOAD_MODAL = "openFileUploadModal";
export const CLOSE_FILE_UPLOAD_MODAL = "closeFileUploadModal";
export const OPEN_MESSAGE_EDIT_MEDIA = "openMessageEditMedia";
export const OPEN_VIEW_FILES_DIALOG = "openViewFiles";
export const SET_EDIT_MESSAGE = "setEditMessageDto";
export const SET_EDIT_MESSAGE_MODAL = "setEditMessageDtoModal";
export const OPEN_EDIT_MESSAGE = "openEditMessage";
export const CLOSE_EDIT_MESSAGE = "closeEditMessage";
export const SET_FILE_ITEM_UUID = "setFileItemUuid";
export const INCREMENT_FILE_ITEM_FILE_COUNT = "incrementFileItemFileCount";
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
export const CHAT_REDRAW = "chatRedraw";
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
export const FILE_UPDATED = "fileUpdated";

export const REACTION_CHANGED = "reactionChanged";

export const REACTION_REMOVED = "reactionRemoved";

export const ATTACH_FILES_TO_MESSAGE_MODAL = "attachFilesToMessage";
export const OPEN_CHAT_EDIT = "openChatEdit";

export const LOAD_FILES_COUNT = "loadFilesCount";

export const OPEN_MESSAGE_READ_USERS_DIALOG = "openMessageReadUsersDialog";

export const FOCUS = "focus";

export const OPEN_PINNED_MESSAGES_MODAL = "openPinnedMessagesModal";

export const OPEN_RESEND_TO_MODAL = "openSendTo";

export const CO_CHATTED_PARTICIPANT_CHANGED = "participantChanged";

export const ADD_VIDEO_SOURCE = "addVideoSource";
export const ADD_SCREEN_SOURCE = "addScreenSource";

export const ADD_VIDEO_SOURCE_DIALOG = "addVideoSourceDialog";

export const SET_LOCAL_MICROPHONE_MUTED = "setLocalMicrophoneMuted";

export const REFRESH_ON_WEBSOCKET_RESTORED = "refreshOnWsRestored";
export const OPEN_PERMISSIONS_WARNING_MODAL = "openPermissionsWarningModal";

export const CHANGE_ROLE_DIALOG = "changeRoleDialog";
