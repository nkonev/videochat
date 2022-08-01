import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const FETCH_USER_PROFILE = 'fetchUserProfile';
export const FETCH_AVAILABLE_OAUTH2_PROVIDERS = 'fetchAvailableOauth2';
export const GET_AVAILABLE_OAUTH2_PROVIDERS = 'getAvailableOauth2';
export const SET_AVAILABLE_OAUTH2_PROVIDERS = 'setAvailableOauth2';
export const GET_SEARCH_STRING = 'getSearchString';
export const SET_SEARCH_STRING = 'setSearchString';
export const UNSET_SEARCH_STRING = 'unsetSearchString';

export const GET_TITLE = 'getTitle';
export const SET_TITLE = 'setTitle';
export const GET_SHOW_SEARCH = 'getShowSearch';
export const SET_SHOW_SEARCH = 'setShowSearch';
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
export const GET_SHOW_CHAT_EDIT_BUTTON = 'getChatEditButton';
export const SET_SHOW_CHAT_EDIT_BUTTON = 'setChatEditButton';

const store = new Vuex.Store({
    state: {
        currentUser: null,
        searchString: null,
        muteVideo: false,
        muteAudio: false,
        title: "",
        isShowSearch: true,
        chatId: null,
        invitedChatId: null,
        chatUsersCount: 0,
        videoChatUsersCount: 0,
        showCallButton: false,
        showHangButton: false,
        shareScreen: false,
        showChatEditButton: false,
        availableOAuth2Providers: []
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
        [SET_VIDEO_CHAT_USERS_COUNT](state, payload) {
            state.videoChatUsersCount = payload;
        },
        [SET_TITLE](state, payload) {
            state.title = payload;
        },
        [SET_SHOW_SEARCH](state, payload) {
            state.isShowSearch = payload;
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
        [GET_VIDEO_CHAT_USERS_COUNT](state) {
            return state.videoChatUsersCount;
        },
        [GET_TITLE](state) {
            return state.title;
        },
        [GET_SHOW_SEARCH](state) {
            return state.isShowSearch;
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
    },
    actions: {
        [FETCH_USER_PROFILE](context) {
            axios.get(`/api/profile`).then(( {data} ) => {
                console.debug("fetched profile =", data);
                context.commit(SET_USER, data);
            });
        },
        [FETCH_AVAILABLE_OAUTH2_PROVIDERS](context) {
            axios.get(`/api/oauth2/providers`).then(( {data} ) => {
                console.debug("fetched oauth2 providers =", data);
                context.commit(SET_AVAILABLE_OAUTH2_PROVIDERS, data);
            });
        },
    }
});

export default store;