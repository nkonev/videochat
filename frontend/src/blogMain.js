import Vue from 'vue'
import BlogApp from './BlogApp.vue'
import vuetify from './plugins/vuetify'
import axios from "axios";
import store, {
    SET_ERROR_COLOR,
    SET_LAST_ERROR,
    SET_SHOW_ALERT,
} from './blogStore';
import router from './blogRouter.js'
import {profile_name} from "@/routes";

let vm;

Vue.prototype.setError = (e, txt, details) => {
    if (details) {
        console.error(txt, e, details);
    } else {
        console.error(txt, e);
    }
    const messageText = e ? (txt + ": " + e) : txt;
    store.commit(SET_LAST_ERROR, messageText);
    store.commit(SET_SHOW_ALERT, true);
    store.commit(SET_ERROR_COLOR, "error");
}

Vue.prototype.setWarning = (txt) => {
    console.warn(txt);
    store.commit(SET_LAST_ERROR, txt);
    store.commit(SET_SHOW_ALERT, true);
    store.commit(SET_ERROR_COLOR, "warning");
}

Vue.prototype.setOk = (txt) => {
    console.info(txt);
    store.commit(SET_LAST_ERROR, txt);
    store.commit(SET_SHOW_ALERT, true);
    store.commit(SET_ERROR_COLOR, "green");
}

Vue.prototype.closeError = () => {
    store.commit(SET_LAST_ERROR, "");
    store.commit(SET_SHOW_ALERT, false);
    store.commit(SET_ERROR_COLOR, "");
}

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
    console.error(consoleErrorMessage);
    const errorMessage  = "Http error. Check the console";
    vm.setError(null, errorMessage);
    return Promise.reject(error)
});

vm = new Vue({
    vuetify,
    store,
    router,
    methods: {

    },
    created(){
        Vue.prototype.isMobile = () => {
            return this.$vuetify.breakpoint.mobile
        };
    },
    watch: {
        '$route': function(newVal, oldVal) {
            if (newVal.name == profile_name) {
                window.location = newVal.fullPath
            }
        }
    },
    // https://ru.vuejs.org/v2/guide/render-function.html
    render: h => h(BlogApp)
}).$mount('#root');
