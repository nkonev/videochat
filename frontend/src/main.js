import Vue from 'vue'
import App from './App.vue'
import vuetify from './plugins/vuetify'
import {setupCentrifuge} from "./centrifugeConnection"
import graphQlClient from "./graphql"
import axios from "axios";
import bus, {
    CHAT_ADD,
    CHAT_DELETED,
    CHAT_EDITED,
    MESSAGE_ADD,
    MESSAGE_DELETED,
    MESSAGE_EDITED,
    UNREAD_MESSAGES_CHANGED,
    USER_PROFILE_CHANGED,
    CHANGE_WEBSOCKET_STATUS,
    LOGGED_OUT,
    LOGGED_IN,
    VIDEO_CALL_INVITED,
    VIDEO_CALL_CHANGED, VIDEO_DIAL_STATUS_CHANGED,
} from './bus';
import store, {
    FETCH_AVAILABLE_OAUTH2_PROVIDERS,
    SET_ERROR_COLOR,
    SET_LAST_ERROR,
    SET_SHOW_ALERT,
    UNSET_USER
} from './store'
import router from './router.js'
import {setIcon} from "@/utils";

let vm;

function getCsrfCookie(name) {
    const value = "; " + document.cookie;
    const parts = value.split("; " + name + "=");
    if (parts.length === 2) return parts.pop().split(";").shift();
}

axios.interceptors.request.use(request => {
    const cookieValue = getCsrfCookie('VIDEOCHAT_XSRF_TOKEN');
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
  if (axios.isCancel(error)) {
    return Promise.reject(error)
  } else if (error && error.response && error.response.status == 401 ) {
    console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
    store.commit(UNSET_USER);
    bus.$emit(LOGGED_OUT, null);
    return Promise.reject(error)
  } else {
    const consoleErrorMessage  = "Request: " + JSON.stringify(error.config) + ", Response: " + JSON.stringify(error.response);
    console.error(consoleErrorMessage);
    const errorMessage  = "Http error. Check the console";
    vm.setError(null, errorMessage);
    return Promise.reject(error)
  }
});

const getData = (message) => {
    return message.data
};

const getProperData = (message) => {
    return message.data.payload
};

const getGlobalEventsData = (message) => {
    return message.data?.globalEvents
};

vm = new Vue({
  vuetify,
  store,
  router,
  methods: {
    connectCentrifuge() {
      this.centrifuge.connect();
    },
    disconnectCentrifuge() {
      this.centrifuge.disconnect();
      Vue.prototype.centrifugeInitialized = false;
    },
    subscribeToGlobalEvents() {
        const onNext = (e) => {
            console.debug("Got event", e);
            if (getGlobalEventsData(e).eventType === 'chat_created') {
                const d = getGlobalEventsData(e).chatEvent;
                bus.$emit(CHAT_ADD, d);
            } else if (getGlobalEventsData(e).eventType === 'chat_edited') {
                const d = getGlobalEventsData(e).chatEvent;
                bus.$emit(CHAT_EDITED, d);
            } else if (getGlobalEventsData(e).eventType === 'chat_deleted') {
                const d = getGlobalEventsData(e).chatEvent;
                bus.$emit(CHAT_DELETED, d);
            }
        }
        const onError = (e) => {
            console.error("Got err, reconnecting", e);
            setTimeout(this.subscribeToGlobalEvents, 2000);
        }
        const onComplete = (e) => {
            console.log("Got compete", e);
        }

        console.log("Subscribing to global events");
        Vue.prototype.globalEventsUnsubscribe = graphQlClient.subscribe(
            {
                query: // ChatDtoWithAdmin
                    `
                    subscription {
                      globalEvents {
                        eventType
                        chatEvent {
                          id
                          name
                          avatar
                          avatarBig
                          lastUpdateDateTime
                          participantIds
                          canEdit
                          canDelete
                          canLeave
                          unreadMessages
                          canBroadcast
                          canVideoKick
                          canChangeChatAdmins
                          tetATet
                          canAudioMute
                          participantsCount
                          changingParticipantsPage
                          participants {
                            id
                            login
                            avatar
                            admin
                          }
                        }
                      }
                    }
                `,
            },
            {
                next: onNext,
                error: onError,
                complete: onComplete,
            },
        );
    },
    unsubscribeFromGlobalEvents() {
        Vue.prototype.globalEventsUnsubscribe();
        Vue.prototype.globalEventsUnsubscribe = null;
    },
  },
  created(){
    Vue.prototype.isMobile = () => {
      return !this.$vuetify.breakpoint.smAndUp
    };
    Vue.prototype.centrifugeInitialized = false;
    const setCentrifugeSession = (cs) => {
      Vue.prototype.centrifugeSessionId = cs;
      bus.$emit(CHANGE_WEBSOCKET_STATUS, {connected: true, wasInitialized: Vue.prototype.centrifugeInitialized});
      Vue.prototype.centrifugeInitialized = true;
    };
    const onDisconnected = () => {
      Vue.prototype.centrifugeSessionId = null;
      bus.$emit(CHANGE_WEBSOCKET_STATUS, {connected: false, wasInitialized: Vue.prototype.centrifugeInitialized});
    };
    Vue.prototype.centrifuge = setupCentrifuge(setCentrifugeSession, onDisconnected);
    this.connectCentrifuge();
    bus.$on(LOGGED_IN, this.connectCentrifuge);
    bus.$on(LOGGED_OUT, this.disconnectCentrifuge);

    this.subscribeToGlobalEvents();
    bus.$on(LOGGED_IN, this.subscribeToGlobalEvents);
    bus.$on(LOGGED_OUT, this.unsubscribeFromGlobalEvents);
  },
  destroyed() {
    this.unsubscribeFromGlobalEvents();
    this.disconnectCentrifuge();
    bus.$off(LOGGED_IN, this.connectCentrifuge);
    bus.$off(LOGGED_OUT, this.disconnectCentrifuge);
  },
  mounted(){
    this.centrifuge.on('publish', (ctx)=>{
      console.debug("Got personal message", ctx);
      if (getData(ctx).type === 'message_created') {
        const d = getProperData(ctx);
        bus.$emit(MESSAGE_ADD, d);
      } else if (getData(ctx).type === 'message_deleted') {
        const d = getProperData(ctx);
        bus.$emit(MESSAGE_DELETED, d);
      } else if (getData(ctx).type === 'message_edited') {
        const d = getProperData(ctx);
        bus.$emit(MESSAGE_EDITED, d);
      } else if (getData(ctx).type === 'unread_messages_changed') {
        const d = getProperData(ctx);
        bus.$emit(UNREAD_MESSAGES_CHANGED, d);
      } else if (getData(ctx).type === 'all_unread_messages_changed') {
          const d = getProperData(ctx);
          const currentNewMessages = d.allUnreadMessages > 0;
          setIcon(currentNewMessages)
      } else if (getData(ctx).type === 'user_profile_changed') {
        const d = getProperData(ctx);
        bus.$emit(USER_PROFILE_CHANGED, d);
      } else if (getData(ctx).type === 'video_call_invitation') {
        const d = getProperData(ctx);
        bus.$emit(VIDEO_CALL_INVITED, d);
      } else if (getData(ctx).type === "video_call_changed") {
        const d = getProperData(ctx);
        bus.$emit(VIDEO_CALL_CHANGED, d);
      } else if (getData(ctx).type === "video_dial_status_changed") {
        const d = getProperData(ctx);
        bus.$emit(VIDEO_DIAL_STATUS_CHANGED, d);
      }

    });

    this.$store.dispatch(FETCH_AVAILABLE_OAUTH2_PROVIDERS);
  },
  // https://ru.vuejs.org/v2/guide/render-function.html
  render: h => h(App)
}).$mount('#root');
