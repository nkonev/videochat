import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex);

export const GET_SEARCH_STRING = 'getSearchString';
export const SET_SEARCH_STRING = 'setSearchString';
export const UNSET_SEARCH_STRING = 'unsetSearchString';
export const GET_SEARCH_NAME = 'getSearchName';
export const SET_SEARCH_NAME = 'setSearchName';
export const GET_SHOW_ALERT = 'getShowAlert';
export const SET_SHOW_ALERT = 'setShowAlert';
export const GET_LAST_ERROR = 'getLastError';
export const SET_LAST_ERROR = 'setLastError';
export const GET_ERROR_COLOR = 'getErrorColor';
export const SET_ERROR_COLOR = 'setErrorColor';
export const GET_SHOW_SEARCH = 'getShowSearch';
export const SET_SHOW_SEARCH = 'setShowSearch';


const store = new Vuex.Store({
    state: {
        searchString: null,
        searchName: null,
        showAlert: false,
        lastError: "",
        errorColor: "",
        isShowSearch: true,
    },
    mutations: {
        [SET_SEARCH_STRING](state, payload) {
            state.searchString = payload;
        },
        [UNSET_SEARCH_STRING](state) {
            state.searchString = "";
        },
        [SET_SEARCH_NAME](state, payload) {
            state.searchName = payload;
        },
        [SET_SHOW_ALERT](state, payload) {
            state.showAlert = payload;
        },
        [SET_ERROR_COLOR](state, payload) {
            state.errorColor = payload;
        },
        [SET_LAST_ERROR](state, payload) {
            state.lastError = payload;
        },
        [SET_SHOW_SEARCH](state, payload) {
            state.isShowSearch = payload;
        },
    },
    getters: {
        [GET_SEARCH_STRING](state) {
            return state.searchString;
        },
        [GET_SEARCH_NAME](state) {
            return state.searchName;
        },
        [GET_SHOW_ALERT](state) {
            return state.showAlert;
        },
        [GET_ERROR_COLOR](state) {
            return state.errorColor;
        },
        [GET_LAST_ERROR](state) {
            return state.lastError;
        },
        [GET_SHOW_SEARCH](state) {
            return state.isShowSearch;
        },
    },
    actions: {

    }
});

export default store;
