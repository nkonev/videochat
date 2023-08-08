/**
 * main.js
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */

// Components
import App from './App.vue'

// Composables
import { createApp } from 'vue'

// Plugins
import {hasLength, isMobileBrowser, offerToJoinToPublicChatStatus} from "@/utils";
import vuetify from "@/plugins/vuetify";
import router from "@/router";
import axios from "axios";
import bus, {LOGGED_OUT} from "@/bus/bus";
import {useChatStore} from "@/store/chatStore";
import pinia from "@/store/index";
import FontAwesomeIcon from "@/plugins/faIcons";

const chatStore = useChatStore();

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

export function registerPlugins (app) {
    app
        .use(vuetify)
        .use(router)
        .use(pinia)
}


const app = createApp(App)

registerPlugins(app)

app.component("font-awesome-icon", FontAwesomeIcon)

app.config.globalProperties.isMobile = () => {
    return isMobileBrowser()
}

app.config.globalProperties.getMessageId = (hash) => {
    if (!hash) {
        return null;
    }
    const str = hash.replace(/\D/g, '');
    return hasLength(str) ? str : null;
};

app.config.globalProperties.setError = (e, txt, details) => {
    if (details) {
        console.error(txt, e, details);
    } else {
        console.error(txt, e);
    }
    const messageText = e ? (txt + ": " + e) : txt;
    chatStore.lastError = messageText;
    chatStore.showAlert = true;
    chatStore.errorColor = "error";
}

app.config.globalProperties.setWarning = (txt) => {
    console.warn(txt);
    chatStore.lastError = txt;
    chatStore.showAlert = true;
    chatStore.errorColor = "warning";
}

app.config.globalProperties.setOk = (txt) => {
    console.info(txt);
    chatStore.lastError = txt;
    chatStore.showAlert = true;
    chatStore.errorColor = "green";
}

app.config.globalProperties.closeError = () => {
    chatStore.lastError = "";
    chatStore.showAlert = false;
    chatStore.errorColor = "";
}

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    // console.log("Catch error", error, error.request, error.response, error.config);
    if (axios.isCancel(error) || error.response.status == offerToJoinToPublicChatStatus) {
        return Promise.reject(error)
    } else if (error && error.response && error.response.status == 401 ) {
        console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
        chatStore.unsetUser();
        bus.emit(LOGGED_OUT, null);
        return Promise.reject(error)
    } else if (!error.config.url.includes('/message/read/')) {
        const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
        console.error(consoleErrorMessage);
        const maybeBusinessMessage = error.response?.data?.message;
        const errorMessage = hasLength(maybeBusinessMessage) ? "Business error" : "Http error. Check the console";
        app.config.globalProperties.setError(maybeBusinessMessage, errorMessage);
        return Promise.reject(error)
    }
});


app.mount('#app')
