import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const GET_PREVIOUS_URL = "getPreviousUrl";
export const SET_PREVIOUS_URL = "setPreviousUrl";
export const UNSET_PREVIOUS_URL = "unsetPreviousUrl";

const store = new Vuex.Store({
    state: {
        currentUser: null,
        previousUrl: ""
    },
    mutations: {
        [UNSET_USER](state) {
            state.currentUser = null;
        },
        [SET_PREVIOUS_URL](state, payload) {
            state.previousUrl = payload;
        },
        [UNSET_PREVIOUS_URL](state) {
            state.previousUrl = "";
        }
    },
    getters: {
        [GET_USER](state) {
            return state.currentUser;
        },
        [GET_PREVIOUS_URL](state) {
            return state.previousUrl;
        },
    },
    actions: {
    }
});

export default store;