import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const FETCH_USER_PROFILE = 'fetchUserProfile';

export const GET_PREVIOUS_URL = "getPreviousUrl";
export const SET_PREVIOUS_URL = "setPreviousUrl";
export const UNSET_PREVIOUS_URL = "unsetPreviousUrl";

const store = new Vuex.Store({
    state: {
        currentUser: null,
        previousUrl: ""
    },
    mutations: {
        [SET_USER](state, payload) {
            console.log("setting user =", payload);
            state.currentUser = payload;
        },
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
            console.log("getting user =", state.currentUser);
            return state.currentUser;
        },
        [GET_PREVIOUS_URL](state) {
            return state.previousUrl;
        },
    },
    actions: {
        [FETCH_USER_PROFILE](context) {
            axios.get(`/api/profile`).then(( {data} ) => {
                console.log("fetched profile =", data);
                context.commit(SET_USER, data);
            });
        },
    }
});

export default store;