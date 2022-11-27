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
    VIDEO_CALL_USER_COUNT_CHANGED,
    VIDEO_DIAL_STATUS_CHANGED,
    PROFILE_SET,
    VIDEO_RECORDING_CHANGED,
    OPEN_SIMPLE_MODAL,
    CLOSE_SIMPLE_MODAL,
} from './bus';
import store, {
    FETCH_AVAILABLE_OAUTH2_PROVIDERS, NOTIFICATION_ADD, NOTIFICATION_DELETE,
    SET_ERROR_COLOR,
    SET_LAST_ERROR,
    SET_SHOW_ALERT,
    UNSET_USER
} from './store'
import router from './router.js'
import {hasLength, setIcon} from "@/utils";
import graphqlSubscriptionMixin from "./graphqlSubscriptionMixin"
import {videochat_name} from "@/routes";

const CheckForNewUrl = '/api/chat/message/check-for-new';

let vm;

function getCookieValue(name) {
    const cookies = document.cookie.split(';');
    const res = cookies.find(c => c.startsWith(name + '='));
    if (res) {
        return res.substring(res.indexOf('=') + 1);
    }
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
  if (axios.isCancel(error)) {
    return Promise.reject(error)
  } else if (error && error.response && error.response.status == 401 ) {
    console.log("Catch 401 Unauthorized, emitting ", LOGGED_OUT);
    store.commit(UNSET_USER);
    bus.$emit(LOGGED_OUT, null);
    return Promise.reject(error)
  } else if (error.config.url != CheckForNewUrl && !error.config.url.includes('/message/read/')) {
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

router.beforeEach((to, from, next) => {
    console.debug("beforeEach", to, vuetify);

    if (from.name == videochat_name && to.name != videochat_name && to.params.leavingVideoAcceptableParam != true) {
        bus.$emit(OPEN_SIMPLE_MODAL, {
            buttonName: vuetify.framework.lang.translator('$vuetify.ok'),
            title: vuetify.framework.lang.translator('$vuetify.leave_call'),
            text: vuetify.framework.lang.translator('$vuetify.leave_call_text'),
            actionFunction: ()=> {
                next();
                bus.$emit(CLOSE_SIMPLE_MODAL);
            },
            cancelFunction: ()=>{
                next(false)
            }
        });
    } else {
        next();
    }
});

vm = new Vue({
  vuetify,
  store,
  router,
  mixins: [graphqlSubscriptionMixin('globalEvents')],
  methods: {
    getGraphQlSubscriptionQuery() {
      return `
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
                    videoUserCountChangedEvent {
                      usersCount
                      chatId
                    }
                    videoRecordingChangedEvent {
                      recordInProgress
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
                    notificationEvent {
                      id
                      chatId
                      messageId
                      notificationType
                      description
                      createDateTime
                    }
                  }
                }
            `
    },
    onNextSubscriptionElement(e) {
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
      } else if (getGlobalEventsData(e).eventType === "video_user_count_changed") {
          const d = getGlobalEventsData(e).videoUserCountChangedEvent;
          bus.$emit(VIDEO_CALL_USER_COUNT_CHANGED, d);
      } else if (getGlobalEventsData(e).eventType === "video_recording_changed") {
          const d = getGlobalEventsData(e).videoRecordingChangedEvent;
          bus.$emit(VIDEO_RECORDING_CHANGED, d);
      } else if (getGlobalEventsData(e).eventType === 'video_call_invitation') {
          const d = getGlobalEventsData(e).videoCallInvitation;
          bus.$emit(VIDEO_CALL_INVITED, d);
      } else if (getGlobalEventsData(e).eventType === "video_dial_status_changed") {
          const d = getGlobalEventsData(e).videoParticipantDialEvent;
          bus.$emit(VIDEO_DIAL_STATUS_CHANGED, d);
      } else if (getGlobalEventsData(e).eventType === 'chat_unread_messages_changed') {
          const d = getGlobalEventsData(e).unreadMessagesNotification;
          bus.$emit(UNREAD_MESSAGES_CHANGED, d);
      } else if (getGlobalEventsData(e).eventType === 'notification_add') {
          const d = getGlobalEventsData(e).notificationEvent;
          store.dispatch(NOTIFICATION_ADD, d);
      } else if (getGlobalEventsData(e).eventType === 'notification_delete') {
          const d = getGlobalEventsData(e).notificationEvent;
          store.dispatch(NOTIFICATION_DELETE, d);
      }
    },
  },
  created(){
    Vue.prototype.isMobile = () => {
      return !this.$vuetify.breakpoint.smAndUp
    };
    Vue.prototype.getHash = (preserveHash) => {
      const tmp = this.$route.hash;
      return preserveHash ? tmp : tmp?.slice(1)
    };
    Vue.prototype.clearHash = () => {
      const hasHash = hasLength(this.getHash());
      if (hasHash) {
          console.debug("Clearing hash");
          const currentRouteName = this.$route.name;
          const routerNewState = {name: currentRouteName};
          routerNewState.query = this.$route.query;
          this.$router.push(routerNewState).catch(() => {
          });
      }
    };
    Vue.prototype.getMessageId = (hash) => {
      return hash.replace(/\D/g, '')
    };

    bus.$on(PROFILE_SET, this.graphQlSubscribe);
    bus.$on(LOGGED_OUT, this.graphQlUnsubscribe);
  },
  destroyed() {
    this.graphQlUnsubscribe();
    graphQlClient.terminate();
    bus.$off(PROFILE_SET, this.graphQlSubscribe);
    bus.$off(LOGGED_OUT, this.graphQlUnsubscribe);
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
  render: h => h(App)
}).$mount('#root');
