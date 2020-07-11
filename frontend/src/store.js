import Vue from 'vue'
import Vuex from 'vuex'
import axios from "axios";

Vue.use(Vuex);

export const GET_USER = 'getUser';
export const SET_USER = 'setUser';
export const UNSET_USER = 'unsetUser';
export const FETCH_USER_PROFILE = 'fetchUserProfile';
export const GET_CENTRIFUGE_SESSION = 'getCentrifugeSession';
export const SET_CENTRIFUGE_SESSION = 'setCentrifugeSession';


const store = new Vuex.Store({
    state: {
        currentUser: null,
        previousUrl: "",
        centrifugeSession: "",
    },
    mutations: {
        [SET_USER](state, payload) {
            state.currentUser = payload;
        },
        [UNSET_USER](state) {
            state.currentUser = null;
        },
        [SET_CENTRIFUGE_SESSION](state, payload) {
            state.centrifugeSession = payload;
        },
    },
    getters: {
        [GET_USER](state) {
            return state.currentUser;
        },
        [GET_CENTRIFUGE_SESSION](state) {
            return state.centrifugeSession;
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