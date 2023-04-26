import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";
import {findIndex, setIcon} from "@/utils";

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const FETCH_USER_PROFILE = 'fetchUserProfile';
export const FETCH_NOTIFICATIONS = 'fetchNotifications';
export const FETCH_AVAILABLE_OAUTH2_PROVIDERS = 'fetchAvailableOauth2';
export const GET_AVAILABLE_OAUTH2_PROVIDERS = 'getAvailableOauth2';
export const SET_AVAILABLE_OAUTH2_PROVIDERS = 'setAvailableOauth2';
export const GET_SEARCH_STRING = 'getSearchString';
export const SET_SEARCH_STRING = 'setSearchString';
export const UNSET_SEARCH_STRING = 'unsetSearchString';

export const GET_TITLE = 'getTitle';
export const SET_TITLE = 'setTitle';
export const GET_AVATAR = "getAvatar";
export const SET_AVATAR = "setAvatar";
export const GET_SHOW_SEARCH = 'getShowSearch';
export const SET_SHOW_SEARCH = 'setShowSearch';
export const GET_SEARCH_NAME = 'getSearchName';
export const SET_SEARCH_NAME = 'setSearchName';
export const GET_CHAT_ID = 'getChatId';
export const SET_CHAT_ID = 'setChatId';
export const GET_CHAT_USERS_COUNT = 'getChatUsesCount';
export const SET_CHAT_USERS_COUNT = 'setChatUsesCount';
export const GET_VIDEO_CHAT_USERS_COUNT = 'getVideoChatUsesCount';
export const SET_VIDEO_CHAT_USERS_COUNT = 'setVideoChatUsesCount';
export const GET_SHOW_CALL_BUTTON = 'getShowCallButton';
export const SET_SHOW_CALL_BUTTON = 'setShowCallButton';
export const GET_SHOW_HANG_BUTTON = 'getShowHangButton';
export const SET_SHOW_HANG_BUTTON = 'setShowHangButton';
export const GET_SHOW_RECORD_START_BUTTON = 'getShowRecordStartButton';
export const SET_SHOW_RECORD_START_BUTTON = 'setShowRecordStartButton';
export const GET_SHOW_RECORD_STOP_BUTTON = 'getShowRecordStopButton';
export const SET_SHOW_RECORD_STOP_BUTTON = 'setShowRecordStopButton';
export const GET_CAN_MAKE_RECORD = 'getCanMakeRecord';
export const SET_CAN_MAKE_RECORD = 'setCanMakeRecord';
export const GET_SHOW_CHAT_EDIT_BUTTON = 'getChatEditButton';
export const SET_SHOW_CHAT_EDIT_BUTTON = 'setChatEditButton';
export const GET_CAN_BROADCAST_TEXT_MESSAGE = 'setCanBroadcastText';
export const SET_CAN_BROADCAST_TEXT_MESSAGE = 'getCanBroadcastText';
export const GET_SHOW_ALERT = 'getShowAlert';
export const SET_SHOW_ALERT = 'setShowAlert';
export const GET_LAST_ERROR = 'getLastError';
export const SET_LAST_ERROR = 'setLastError';
export const GET_ERROR_COLOR = 'getErrorColor';
export const SET_ERROR_COLOR = 'setErrorColor';
export const GET_NOTIFICATIONS = 'getNotifications';
export const SET_NOTIFICATIONS = 'setNotifications';
export const UNSET_NOTIFICATIONS = 'unsetNotifications';
export const GET_NOTIFICATIONS_SETTINGS = 'getNotificationsSettings';
export const SET_NOTIFICATIONS_SETTINGS = 'setNotificationsSettings';
export const GET_SHOULD_PHONE_BLINK = 'getShouldPhoneBlink';
export const SET_SHOULD_PHONE_BLINK = 'setShouldPhoneBlink';
export const GET_TET_A_TET = 'getTetATet';
export const SET_TET_A_TET = 'setTetATet';
export const NOTIFICATION_ADD = 'notificationAdd';
export const NOTIFICATION_DELETE = 'notificationDelete';
export const GET_SHOW_MICROPHONE_ON_BUTTON = 'getShowMicroOn';
export const SET_SHOW_MICROPHONE_ON_BUTTON = 'setShowMicroOn';
export const GET_SHOW_MICROPHONE_OFF_BUTTON = 'getShowMicroOff';
export const SET_SHOW_MICROPHONE_OFF_BUTTON = 'setShowMicroOff';
export const GET_CAN_SHOW_MICROPHONE_BUTTON = 'getCanShowMicro';
export const SET_CAN_SHOW_MICROPHONE_BUTTON = 'setCanShowMicro';
export const GET_INITIALIZING_STARTING_VIDEO_RECORD = 'getInitializingStaringVideoRecord';
export const SET_INITIALIZING_STARTING_VIDEO_RECORD = 'setInitializingStaringVideoRecord';
export const GET_INITIALIZING_STOPPING_VIDEO_RECORD = 'getInitializingStoppingVideoRecord';
export const SET_INITIALIZING_STOPPING_VIDEO_RECORD = 'setInitializingStoppingVideoRecord';

const store = new Vuex.Store({
    state: {
        currentUser: null,
        searchString: null,
        muteVideo: false,
        muteAudio: false,
        title: "",
        avatar: null,
        isShowSearch: true,
        searchName: null,
        chatId: null,
        invitedChatId: null,
        chatUsersCount: 0,
        videoChatUsersCount: 0,
        showCallButton: false,
        showHangButton: false,
        showRecordStartButton: false,
        showRecordStopButton: false,
        canMakeRecord: false,
        shareScreen: false,
        showChatEditButton: false,
        availableOAuth2Providers: [],
        canBroadcastTextMessage: false,
        showAlert: false,
        lastError: "",
        errorColor: "",
        notifications: [],
        notificationsSettings: {},
        shouldPhoneBlink: false,
        tetATet: false,
        showMicrophoneOnButton: false,
        showMicrophoneOffButton: false,
        canShowMicrophoneButton: false,
        initializingStaringVideoRecord: false,
        initializingStoppingVideoRecord: false,
    },
    mutations: {
        [SET_USER](state, payload) {
            state.currentUser = payload;
        },
        [SET_SEARCH_STRING](state, payload) {
            state.searchString = payload;
        },
        [UNSET_USER](state) {
            state.currentUser = null;
        },
        [UNSET_SEARCH_STRING](state) {
            state.searchString = "";
        },
        [SET_SHOW_CALL_BUTTON](state, payload) {
            state.showCallButton = payload;
        },
        [SET_SHOW_HANG_BUTTON](state, payload) {
            state.showHangButton = payload;
        },
        [SET_SHOW_RECORD_START_BUTTON](state, payload) {
            state.showRecordStartButton = payload;
        },
        [SET_SHOW_RECORD_STOP_BUTTON](state, payload) {
            state.showRecordStopButton = payload;
        },
        [SET_CAN_MAKE_RECORD](state, payload) {
            state.canMakeRecord = payload;
        },
        [SET_VIDEO_CHAT_USERS_COUNT](state, payload) {
            state.videoChatUsersCount = payload;
        },
        [SET_TITLE](state, payload) {
            state.title = payload;
        },
        [SET_AVATAR](state, payload) {
            state.avatar = payload;
        },
        [SET_SHOW_SEARCH](state, payload) {
            state.isShowSearch = payload;
        },
        [SET_SEARCH_NAME](state, payload) {
            state.searchName = payload;
        },
        [SET_CHAT_USERS_COUNT](state, payload) {
            state.chatUsersCount = payload;
        },
        [SET_SHOW_CHAT_EDIT_BUTTON](state, payload) {
            state.showChatEditButton = payload;
        },
        [SET_CHAT_ID](state, payload) {
            state.chatId = payload;
        },
        [SET_AVAILABLE_OAUTH2_PROVIDERS](state, payload) {
            state.availableOAuth2Providers = payload;
        },
        [SET_CAN_BROADCAST_TEXT_MESSAGE](state, payload) {
            state.canBroadcastTextMessage = payload;
        },
        [SET_SHOW_ALERT](state, payload) {
            state.showAlert = payload;
        },
        [SET_LAST_ERROR](state, payload) {
            state.lastError = payload;
        },
        [SET_ERROR_COLOR](state, payload) {
            state.errorColor = payload;
        },
        [SET_NOTIFICATIONS](state, payload) {
            state.notifications = payload;
            setIcon(payload != null && payload.length > 0);
        },
        [UNSET_NOTIFICATIONS](state, payload) {
            state.notifications = [];
            setIcon(false);
        },
        [SET_NOTIFICATIONS_SETTINGS](state, payload) {
            state.notificationsSettings = payload;
        },
        [SET_SHOULD_PHONE_BLINK](state, payload) {
            state.shouldPhoneBlink = payload;
        },
        [SET_TET_A_TET](state, payload) {
            state.tetATet = payload;
        },
        [SET_SHOW_MICROPHONE_ON_BUTTON](state, payload) {
            state.showMicrophoneOnButton = payload;
        },
        [SET_SHOW_MICROPHONE_OFF_BUTTON](state, payload) {
            state.showMicrophoneOffButton = payload;
        },
        [SET_CAN_SHOW_MICROPHONE_BUTTON](state, payload) {
            state.canShowMicrophoneButton = payload;
        },
        [SET_INITIALIZING_STARTING_VIDEO_RECORD](state, payload) {
            state.initializingStaringVideoRecord = payload;
        },
        [SET_INITIALIZING_STOPPING_VIDEO_RECORD](state, payload) {
            state.initializingStoppingVideoRecord = payload;
        },
    },
    getters: {
        [GET_USER](state) {
            return state.currentUser;
        },
        [GET_SEARCH_STRING](state) {
            return state.searchString;
        },
        [GET_SHOW_CALL_BUTTON](state) {
            return state.showCallButton;
        },
        [GET_SHOW_HANG_BUTTON](state) {
            return state.showHangButton;
        },
        [GET_SHOW_RECORD_START_BUTTON](state) {
            return state.showRecordStartButton;
        },
        [GET_SHOW_RECORD_STOP_BUTTON](state) {
            return state.showRecordStopButton;
        },
        [GET_CAN_MAKE_RECORD](state) {
            return state.canMakeRecord;
        },
        [GET_VIDEO_CHAT_USERS_COUNT](state) {
            return state.videoChatUsersCount;
        },
        [GET_TITLE](state) {
            return state.title;
        },
        [GET_AVATAR](state) {
            return state.avatar;
        },
        [GET_SHOW_SEARCH](state) {
            return state.isShowSearch;
        },
        [GET_SEARCH_NAME](state) {
            return state.searchName;
        },
        [GET_CHAT_USERS_COUNT](state) {
            return state.chatUsersCount;
        },
        [GET_SHOW_CHAT_EDIT_BUTTON](state) {
            return state.showChatEditButton;
        },
        [GET_CHAT_ID](state) {
            return state.chatId;
        },
        [GET_AVAILABLE_OAUTH2_PROVIDERS](state) {
            return state.availableOAuth2Providers;
        },
        [GET_CAN_BROADCAST_TEXT_MESSAGE](state) {
            return state.canBroadcastTextMessage;
        },
        [GET_SHOW_ALERT](state) {
            return state.showAlert;
        },
        [GET_LAST_ERROR](state) {
            return state.lastError;
        },
        [GET_ERROR_COLOR](state) {
            return state.errorColor;
        },
        [GET_NOTIFICATIONS](state) {
            return state.notifications;
        },
        [GET_NOTIFICATIONS_SETTINGS](state) {
            return state.notificationsSettings;
        },
        [GET_SHOULD_PHONE_BLINK](state) {
            return state.shouldPhoneBlink;
        },
        [GET_TET_A_TET](state) {
            return state.tetATet;
        },
        [GET_SHOW_MICROPHONE_ON_BUTTON](state) {
            return state.showMicrophoneOnButton;
        },
        [GET_SHOW_MICROPHONE_OFF_BUTTON](state) {
            return state.showMicrophoneOffButton;
        },
        [GET_CAN_SHOW_MICROPHONE_BUTTON](state) {
            return state.canShowMicrophoneButton;
        },
        [GET_INITIALIZING_STARTING_VIDEO_RECORD](state) {
            return state.initializingStaringVideoRecord;
        },
        [GET_INITIALIZING_STOPPING_VIDEO_RECORD](state) {
            return state.initializingStoppingVideoRecord;
        },
    },
    actions: {
        [FETCH_USER_PROFILE](context) {
            axios.get(`/api/profile`).then(( {data} ) => {
                console.debug("fetched profile =", data);
                context.commit(SET_USER, data);
            });
        },
        [FETCH_AVAILABLE_OAUTH2_PROVIDERS](context) {
            return axios.get(`/api/oauth2/providers`).then(( {data} ) => {
                console.debug("fetched oauth2 providers =", data);
                context.commit(SET_AVAILABLE_OAUTH2_PROVIDERS, data);
            });
        },
        [FETCH_NOTIFICATIONS](context) {
            axios.get(`/api/notification/notification`).then(( {data} ) => {
                console.debug("fetched notifications =", data);
                context.commit(SET_NOTIFICATIONS, data);
            });
            axios.get(`/api/notification/settings`).then(( {data} ) => {
                console.debug("fetched notifications settings =", data);
                context.commit(SET_NOTIFICATIONS_SETTINGS, data);
            });
        },
        [NOTIFICATION_ADD](context, payload) {
            const newArr = [payload, ...context.state.notifications];
            context.commit(SET_NOTIFICATIONS, newArr);
        },
        [NOTIFICATION_DELETE](context, payload) {
            const newArr = context.state.notifications;
            const idxToRemove = findIndex(newArr, payload);
            newArr.splice(idxToRemove, 1);
            context.commit(SET_NOTIFICATIONS, newArr);
        },
    }
});

export default store;
