import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";
import {audioMuteDefault} from "./utils";

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const FETCH_USER_PROFILE = 'fetchUserProfile';
export const GET_SEARCH_STRING = 'getSearchString';
export const SET_SEARCH_STRING = 'setSearchString';
export const CHANGE_SEARCH_STRING = 'changeSearchString';
export const GET_MUTE_VIDEO = 'getMuteVideo';
export const SET_MUTE_VIDEO = 'setMuteVideo';
export const GET_MUTE_AUDIO = 'getMuteAudio';
export const SET_MUTE_AUDIO = 'setMuteAudio';

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
export const GET_SHARE_SCREEN = 'getShareScreenButton';
export const SET_SHARE_SCREEN = 'setShareScreenButton';
export const GET_SHOW_CHAT_EDIT_BUTTON = 'getChatEditButton';
export const SET_SHOW_CHAT_EDIT_BUTTON = 'setChatEditButton';

const store = new Vuex.Store({
    state: {
        currentUser: null,
        searchString: "",
        muteVideo: false,
        muteAudio: audioMuteDefault,
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
    },
    mutations: {
        [SET_USER](state, payload) {
            state.currentUser = payload;
        },
        [UNSET_USER](state) {
            state.currentUser = null;
        },
        [SET_SEARCH_STRING](state, payload) {
            state.searchString = payload;
        },
        [SET_MUTE_VIDEO](state, payload) {
            state.muteVideo = payload;
        },
        [SET_MUTE_AUDIO](state, payload) {
            state.muteAudio = payload;
        },
        [SET_SHOW_CALL_BUTTON](state, payload) {
            state.showCallButton = payload;
        },
        [SET_SHOW_HANG_BUTTON](state, payload) {
            state.showHangButton = payload;
        },
        [SET_SHARE_SCREEN](state, payload) {
            state.shareScreen = payload;
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
    },
    getters: {
        [GET_USER](state) {
            return state.currentUser;
        },
        [GET_SEARCH_STRING](state) {
            return state.searchString;
        },
        [GET_MUTE_VIDEO](state) {
            return state.muteVideo;
        },
        [GET_MUTE_AUDIO](state) {
            return state.muteAudio;
        },
        [GET_SHOW_CALL_BUTTON](state) {
            return state.showCallButton;
        },
        [GET_SHOW_HANG_BUTTON](state) {
            return state.showHangButton;
        },
        [GET_SHARE_SCREEN](state) {
            return state.shareScreen;
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
    },
    actions: {
        [FETCH_USER_PROFILE](context) {
            axios.get(`/api/profile`).then(( {data} ) => {
                console.debug("fetched profile =", data);
                context.commit(SET_USER, data);
            });
        },
    }
});

export default store;