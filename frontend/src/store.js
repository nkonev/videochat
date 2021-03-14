import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";
import bus, {CHAT_SEARCH_CHANGED} from "./bus";

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


const store = new Vuex.Store({
    state: {
        currentUser: null,
        searchString: "",
        muteVideo: false,
        muteAudio: false
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
    },
    actions: {
        [FETCH_USER_PROFILE](context) {
            axios.get(`/api/profile`).then(( {data} ) => {
                console.debug("fetched profile =", data);
                context.commit(SET_USER, data);
            });
        },
        [CHANGE_SEARCH_STRING](context, data) {
            context.commit(SET_SEARCH_STRING, data);
            bus.$emit(CHAT_SEARCH_CHANGED);
        },

    }
});

export default store;