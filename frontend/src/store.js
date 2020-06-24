import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex);

export const GET_UNAUTHENTICATED = 'getUnauthenticated';
export const SET_UNAUTHENTICATED = 'setUnauthenticated';

const store = new Vuex.Store({
    state: {
        unauthenticated: false,
    },
    mutations: {
        [SET_UNAUTHENTICATED](state, payload) {
            state.unauthenticated = payload;
        },
    },
    getters: {
        [GET_UNAUTHENTICATED](state) {
            return state.unauthenticated;
        },
    },
    actions: {
    }
});

export default store;