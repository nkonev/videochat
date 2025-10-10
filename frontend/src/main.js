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
import {getIdFromRouteHash, hasLength, isMobileBrowser} from "@/utils";
import vuetify from "@/plugins/vuetify";
import router from "@/router";
import axios from "axios";
import bus, {LOGGED_OUT} from "@/bus/bus";
import {useChatStore} from "@/store/chatStore";
import pinia from "@/store/index";
import FontAwesomeIcon from "@/plugins/faIcons";
import debounce from "lodash/debounce.js";

axios.defaults.xsrfCookieName = "VIDEOCHAT_XSRF_TOKEN";
axios.defaults.xsrfHeaderName = "X-XSRF-TOKEN";

const webSplitpanesCss = () => import('@/splitpanesWeb.css');
const mobileSplitpanesCss = () => import("@/splitpanesMobile.scss");

// it's placed here, before app creation
// otherwise, if we put it into ChatView.created()
// it is going to break MessageList.scrollTo()
// to check -
// 1. scroll to certain message
// 2. reload the page
// 3. top message should be the same after page reloading
if (isMobileBrowser()) {
  mobileSplitpanesCss()
} else {
  webSplitpanesCss()
}

const chatStore = useChatStore();

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

app.config.globalProperties.getIdFromRouteHash = getIdFromRouteHash;

app.config.globalProperties.setError = (e, txt, traceId) => {
    console.error(txt, e, "traceId=", traceId);
    let messageText = "";
    messageText += txt;
    if (e) {
      messageText += ": ";
      messageText += e;
    }
    if (traceId) {
      messageText += ", traceId=";
      messageText += traceId;
    }
    chatStore.lastError = messageText;
    chatStore.showAlert = true;
    chatStore.showAlertDebounced = true;
    chatStore.errorColor = "error";
}

// Not to show error on websocket connection interruptions
app.config.globalProperties.setErrorSilent = (e, traceId) => {
    console.error(e, "traceId=", traceId);
}

app.config.globalProperties.setWarning = (txt) => {
    console.warn(txt);
    chatStore.lastError = txt;
    chatStore.showAlert = true;
    chatStore.showAlertDebounced = true;
    chatStore.errorColor = "warning";
}

app.config.globalProperties.setOk = (txt) => {
    console.info(txt);
    chatStore.lastError = txt;
    chatStore.showAlert = true;
    chatStore.showAlertDebounced = true;
    chatStore.errorColor = "green";
}

app.config.globalProperties.setTempNotification = (txt) => {
    console.info(txt);
    chatStore.lastError = txt;
    chatStore.showAlert = true;
    chatStore.showAlertDebounced = true;
    chatStore.errorColor = "black";
    chatStore.alertTimeout = 3000;
}

app.config.globalProperties.setTempGoTo = (txt, actionText, action) => {
    console.info(txt);
    chatStore.showTempGoTo = txt;
    chatStore.showAlert = true;
    chatStore.showAlertDebounced = true;
    chatStore.errorColor = "black";
    chatStore.alertTimeout = 5000;
}

// fixes https://stackoverflow.com/questions/49627750/vuetify-closing-snackbar-without-closing-dialog
// testcase
// user 1 enters creates a video call
// user 2 from the different chat tries to call user 1
// user 2 will have an orange snackbar "user 1 is busy"
// user 2 closes the snackbar
// ChatParticipantsModal shouldn't disappear for user 2
const hideAlert = () => {
    chatStore.showAlertDebounced = false;
}

const debouncedHideAlert = debounce(hideAlert, 3000);

app.config.globalProperties.closeError = () => {
    chatStore.lastError = "";
    chatStore.showAlert = false;
    debouncedHideAlert();
    chatStore.errorColor = "";
}

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    // console.log("Catch error", error, error.request, error.response, error.config);
    if (axios.isCancel(error)) {
        return Promise.reject(error)
    } else if (error && error.response && error.response.status == 401 ) {
        console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
        chatStore.unsetUser();
        bus.emit(LOGGED_OUT);
        return Promise.reject(error)
    } else if (error.code == 'ECONNABORTED') { // removes error snackbar caused by cancelled message read request
        console.warn("Connection aborted")
        return Promise.reject(error)
    } else if (error?.config?.url?.endsWith('git.json')) {
        return Promise.reject(error)
    } else if (error.config.url == '/api/aaa/ping') { // removes error snackbar caused by ping
        return Promise.reject(error)
        // removes error snackbar caused by wrong password in
    } else if (error.response.status == 400 && (
            error.config.url.match(/\/api\/aaa\/user\/\d+\/password/) || // SetPasswordModal.vue
            error.config.url == "/api/aaa/password-reset-set-new" ||
            error.config.url == "/api/aaa/register" ||
            error.config.url == "/api/aaa/profile"
    )) {
        return Promise.reject(error)
    } else {
        const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
        console.error(consoleErrorMessage);
        let maybeBusinessMessage = error.response?.data?.message;
        let errorMessage;
        if (hasLength(maybeBusinessMessage)) {
            errorMessage = "Business error";
        } else {
            errorMessage = "Http error. Check the console";
            maybeBusinessMessage = "";
        }
        const respHeaders = error.response?.headers;
        let traceId;
        if (respHeaders) {
          traceId = respHeaders['x-traceid']
        }
        const methodUrl = "" + error.config?.method + " " + error.config?.url;
        if (error.response) {
            app.config.globalProperties.setError(methodUrl + " " + maybeBusinessMessage, errorMessage, traceId);
        } else {
            app.config.globalProperties.setErrorSilent(methodUrl + " " + errorMessage, traceId);
        }
        return Promise.reject(error)
    }
});

app.mount('#app')
