import Vue from 'vue'
import BlogApp from './BlogApp.vue'
import vuetify from './plugins/vuetify'
import axios from "axios";
import bus, {
    LOGGED_OUT,
    PROFILE_SET,
} from './bus';
import store, {
    SET_ERROR_COLOR,
    SET_LAST_ERROR,
    SET_SHOW_ALERT,
    UNSET_USER
} from './store'
import router from './blogRouter.js'

let vm;

function getCookieValue(name) {
    const value = "; " + document.cookie;
    const parts = value.split("; " + name + "=");
    if (parts.length === 2) return parts.pop().split(";").shift();
}

axios.interceptors.request.use(request => {
    const cookieValue = getCookieValue('VIDEOCHAT_XSRF_TOKEN');
    console.debug("Injecting xsrf token to header", cookieValue);
    request.headers['X-XSRF-TOKEN'] = cookieValue;
    return request
})

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
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    // console.log("Catch error", error, error.request, error.response, error.config);
    if (axios.isCancel(error) || error.response.status == 417) {
        return Promise.reject(error)
    } else if (error && error.response && error.response.status == 401 ) {
        console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
        store.commit(UNSET_USER);
        bus.$emit(LOGGED_OUT, null);
        return Promise.reject(error)
    } else if (!error.config.url.includes('/message/read/')) {
        const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
        console.error(consoleErrorMessage);
        const errorMessage  = "Http error. Check the console";
        vm.setError(null, errorMessage);
        return Promise.reject(error)
    }
});

vm = new Vue({
    vuetify,
    store,
    router,
    methods: {

    },
    created(){
        Vue.prototype.isMobile = () => {
            return !this.$vuetify.breakpoint.smAndUp
        };
    },
    destroyed() {
    },
    mounted(){

    },
    watch: {
        '$store.state.currentUser': function(newUserValue, oldUserValue) {
            console.debug("User new", newUserValue, "old" , oldUserValue);
            if (newUserValue && !oldUserValue) {
                bus.$emit(PROFILE_SET);
            }
        }
    },
    // https://ru.vuejs.org/v2/guide/render-function.html
    render: h => h(BlogApp)
}).$mount('#root');
