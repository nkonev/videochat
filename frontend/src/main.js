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
import {getUrlPrefix, hasLength, isMobileBrowser} from "@/utils";
import vuetify from "@/plugins/vuetify";
import router from "@/router";
import axios from "axios";
import bus, {LOGGED_OUT} from "@/bus/bus";
import {useChatStore} from "@/store/chatStore";
import pinia from "@/store/index";
import FontAwesomeIcon from "@/plugins/faIcons";

axios.defaults.xsrfCookieName = "VIDEOCHAT_XSRF_TOKEN";
axios.defaults.xsrfHeaderName = "X-XSRF-TOKEN";

const webSplitpanesCss = () => import('splitpanes/dist/splitpanes.css');
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

const check = () => {
  if (!('serviceWorker' in navigator)) {
    throw new Error('No Service Worker support!')
  }
}

const registerServiceWorker = async () => {
  const swRegistration = await navigator.serviceWorker.register(getUrlPrefix() + '/service.js'); //notice the file name
  return swRegistration;
}
const requestNotificationPermission = async () => {
  const permission = await window.Notification.requestPermission();
  // value of permission can be 'granted', 'default', 'denied'
  // granted: user has accepted the request
  // default: user has dismissed the notification permission popup by clicking on x
  // denied: user has denied the request.
  if(permission !== 'granted'){
    throw new Error('Permission not granted for Notification');
  }
}
check();
const swRegistration = await registerServiceWorker();
const permission =  await requestNotificationPermission();

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

app.config.globalProperties.getMessageId = (hash) => {
    if (!hash) {
        return null;
    }
    const str = hash.replace(/\D/g, '');
    return hasLength(str) ? str : null;
};

app.config.globalProperties.setError = (e, txt, traceId) => {
    console.error(txt, e);
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
    if (axios.isCancel(error)) {
        return Promise.reject(error)
    } else if (error && error.response && error.response.status == 401 ) {
        console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
        chatStore.unsetUser();
        bus.emit(LOGGED_OUT, null);
        return Promise.reject(error)
    } else {
        const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
        console.error(consoleErrorMessage);
        const maybeBusinessMessage = error.response?.data?.message;
        const errorMessage = hasLength(maybeBusinessMessage) ? "Business error" : "Http error. Check the console";
        const respHeaders = error.response?.headers;
        let traceId;
        if (respHeaders) {
          traceId = respHeaders['trace-id']
        }
        app.config.globalProperties.setError(maybeBusinessMessage, errorMessage, traceId);
        return Promise.reject(error)
    }
});


app.mount('#app')
