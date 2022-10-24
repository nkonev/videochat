import Vue from 'vue'
import App from './App.vue'
import vuetify from './plugins/vuetify'
import graphQlClient from "./graphql"
import axios from "axios";
import bus, {
    CHAT_ADD,
    CHAT_DELETED,
    CHAT_EDITED,
    UNREAD_MESSAGES_CHANGED,
    USER_PROFILE_CHANGED,
    LOGGED_OUT,
    LOGGED_IN,
    VIDEO_CALL_INVITED,
    VIDEO_CALL_CHANGED, VIDEO_DIAL_STATUS_CHANGED, PROFILE_SET,
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

const getGlobalEventsData = (message) => {
    return message.data?.globalEvents
};

vm = new Vue({
  vuetify,
  store,
  router,
  methods: {
    subscribeToGlobalEvents() {
        const onNext = (e) => {
            console.debug("Got global event", e);
            if (e.errors != null && e.errors.length) {
                this.setError(null, "Error in globalEvents subscription");
            }
            if (getGlobalEventsData(e).eventType === 'chat_created') {
                const d = getGlobalEventsData(e).chatEvent;
                bus.$emit(CHAT_ADD, d);
            } else if (getGlobalEventsData(e).eventType === 'chat_edited') {
                const d = getGlobalEventsData(e).chatEvent;
                bus.$emit(CHAT_EDITED, d);
            } else if (getGlobalEventsData(e).eventType === 'chat_deleted') {
                const d = getGlobalEventsData(e).chatDeletedEvent;
                bus.$emit(CHAT_DELETED, d);
            } else if (getGlobalEventsData(e).eventType === 'user_profile_changed') {
                const d = getGlobalEventsData(e).userEvent;
                bus.$emit(USER_PROFILE_CHANGED, d);
            } else if (getGlobalEventsData(e).eventType === "video_call_changed") {
                const d = getGlobalEventsData(e).videoEvent;
                bus.$emit(VIDEO_CALL_CHANGED, d);
            } else if (getGlobalEventsData(e).eventType === 'video_call_invitation') {
                const d = getGlobalEventsData(e).videoCallInvitation;
                bus.$emit(VIDEO_CALL_INVITED, d);
            } else if (getGlobalEventsData(e).eventType === "video_dial_status_changed") {
                const d = getGlobalEventsData(e).videoParticipantDialEvent;
                bus.$emit(VIDEO_DIAL_STATUS_CHANGED, d);
            } else if (getGlobalEventsData(e).eventType === 'chat_unread_messages_changed') {
                const d = getGlobalEventsData(e).unreadMessagesNotification;
                bus.$emit(UNREAD_MESSAGES_CHANGED, d);
            } else if (getGlobalEventsData(e).eventType === 'all_unread_messages_changed') {
                const d = getGlobalEventsData(e).allUnreadMessagesNotification;
                const currentNewMessages = d.allUnreadMessages > 0;
                setIcon(currentNewMessages)
            }
        }
        const onError = (e) => {
            console.error("Got err in global event subscription, reconnecting", e);
            setTimeout(this.subscribeToGlobalEvents, 2000);
        }
        const onComplete = () => {
            console.log("Got compete in global event subscription");
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
                        chatDeletedEvent {
                          id
                        }
                        userEvent {
                          id
                          login
                          avatar
                        }
                        videoEvent {
                          usersCount
                          chatId
                        }
                        videoCallInvitation {
                          chatId
                          chatName
                        }
                        videoParticipantDialEvent {
                          chatId
                          dials {
                            userId
                            status
                          }
                        }
                        unreadMessagesNotification {
                          chatId
                          unreadMessages
                        }
                        allUnreadMessagesNotification {
                          allUnreadMessages
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

        axios.put(`/api/chat/message/check-for-new`).then(({data}) => {
            console.debug("New messages response", data);
            if (data) {
                const currentNewMessages = data.allUnreadMessages > 0;
                setIcon(currentNewMessages)
            }
        })
    },
    unsubscribeFromGlobalEvents() {
        console.log("Unsubscribing from global events");
        if (Vue.prototype.globalEventsUnsubscribe) {
            Vue.prototype.globalEventsUnsubscribe();
        }
        Vue.prototype.globalEventsUnsubscribe = null;
    },
  },
  created(){
    Vue.prototype.isMobile = () => {
      return !this.$vuetify.breakpoint.smAndUp
    };
    bus.$on(PROFILE_SET, this.subscribeToGlobalEvents);
    bus.$on(LOGGED_OUT, this.unsubscribeFromGlobalEvents);
  },
  destroyed() {
    this.unsubscribeFromGlobalEvents();
    graphQlClient.terminate();
    bus.$off(PROFILE_SET, this.subscribeToGlobalEvents);
    bus.$off(LOGGED_OUT, this.unsubscribeFromGlobalEvents);
  },
  mounted(){
    this.$store.dispatch(FETCH_AVAILABLE_OAUTH2_PROVIDERS);
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
  render: h => h(App)
}).$mount('#root');
