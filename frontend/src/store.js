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

const store = new Vuex.Store({
    state: {
        currentUser: null,
        searchString: ""
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
    },
    getters: {
        [GET_USER](state) {
            return state.currentUser;
        },
        [GET_SEARCH_STRING](state) {
            return state.searchString;
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