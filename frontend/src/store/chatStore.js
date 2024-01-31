// Utilities
import { defineStore } from 'pinia'
import axios from "axios";
import {isMobileBrowser, setIcon} from "@/utils";
import {SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";

export const callStateReady = "ready"
export const callStateInCall = "inCall"

export const useChatStore = defineStore('chat', {
  state: () => {
    return {
        currentUser: null,
        notificationsCount: 0,
        notificationsSettings: {},
        showCallManagement: false,
        callState: callStateReady,
        shouldPhoneBlink: false,
        availableOAuth2Providers: [],
        showAlert: false,
        lastError: "",
        errorColor: "",
        showDrawer: !isMobileBrowser(),
        showDrawerPrevious: false,
        isShowSearch: false,
        searchType: SEARCH_MODE_CHATS,
        showScrollDown: false,
        title: "",
        avatar: null,
        chatUsersCount: 0,
        showChatEditButton: false,
        canBroadcastTextMessage: false,
        tetATet: false,
        showGoToBlogButton: null,
        videoChatUsersCount: 0,
        canMakeRecord: false,
        showRecordStartButton: false,
        showRecordStopButton: false,
        progressCount: 0,
        fileUploadingQueue: [],
        moreImportantSubtitleInfo: null,
        initializingStaringVideoRecord: false,
        initializingStoppingVideoRecord: false,
        canShowMicrophoneButton: false,
        showMicrophoneOnButton: false,
        showMicrophoneOffButton: false,
        leavingVideoAcceptableParam: false,
        initializingVideoCall: false,
        isEditingBigText: false,
    }
  },
  actions: {
    unsetUser() {
      this.currentUser = null
    },
    fetchUserProfile() {
        axios.get(`/api/aaa/profile`).then(( {data} ) => {
            console.debug("fetched profile =", data);
            this.currentUser = data;
        });
    },
    fetchAvailableOauth2Providers() {
          return axios.get(`/api/aaa/oauth2/providers`).then(( {data} ) => {
              console.debug("fetched oauth2 providers =", data);
              this.availableOAuth2Providers = data;
          });
    },
    setNotificationCount(count){
      this.notificationsCount = count;
      setIcon(count > 0);
    },
    fetchNotificationsCount() {
      axios.get(`/api/notification/count`).then(( {data} ) => {
        console.debug("fetched notifications =", data);
        this.setNotificationCount(data.totalCount);
      });
      axios.get(`/api/notification/settings`).then(( {data} ) => {
        console.debug("fetched notifications settings =", data);
        this.notificationsSettings = data;
      });
    },
    unsetNotifications() {
      this.notificationsCount = 0;
      setIcon(false);
    },
    switchSearchType() {
      if (this.searchType == SEARCH_MODE_CHATS) {
        this.searchType = SEARCH_MODE_MESSAGES
      } else if (this.searchType == SEARCH_MODE_MESSAGES) {
        this.searchType = SEARCH_MODE_CHATS
      }
    },
    incrementProgressCount() {
      this.progressCount++
    },
    decrementProgressCount() {
      if (this.progressCount > 0) {
        this.progressCount--
      } else {
        const err = new Error();
        console.warn("Attempt to decrement progressCount lower than 0", err.stack)
      }
    },
    appendToFileUploadingQueue(aFile) {
        this.fileUploadingQueue.push(aFile)
    },
    removeFromFileUploadingQueue(id) {
        this.fileUploadingQueue = this.fileUploadingQueue.filter((item) => {
            return item.id != id;
        });
    },
    isInCall() {
      return this.callState == callStateInCall
    },
    isReady() {
      return this.callState == callStateReady
    },
    setCallStateReady() {
      this.callState = callStateReady
    },
    setCallStateInCall() {
      this.callState = callStateInCall
    },
  },

})
