// Utilities
import { defineStore } from 'pinia'
import axios from "axios";
import {isMobileBrowser, setIcon} from "@/utils";
import {SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import {setStoredLanguage} from "@/store/localStore";
import bus, {PROFILE_SET} from "@/bus/bus.js";

export const callStateReady = "ready"
export const callStateInCall = "inCall"
export const fileUploadingSessionTypeMessageEdit = "fromMessageEdit"
export const fileUploadingSessionTypeMedia = "media"

const chatDtoFactory = () => {
    return {
        participantIds:[],
        participants:[],
    }
}

export const useChatStore = defineStore('chat', {
  state: () => {
    return {
        currentUser: null,
        notificationsCount: 0,
        showCallManagement: false,
        callState: callStateReady,
        shouldPhoneBlink: false,
        availableOAuth2Providers: [],
        OAuth2ProvidersAllowUnbind: {},
        showAlert: false,
        showAlertDebounced: false,
        alertTimeout: null,
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
        fileUploadingSessionType: null,
        moreImportantSubtitleInfo: null,
        usersWritingSubtitleInfo: null,
        initializingStaringVideoRecord: false,
        initializingStoppingVideoRecord: false,
        canShowMicrophoneButton: false,
        localMicrophoneEnabled: false, // current state - enabled or not
        canShowVideoButton: false,
        localVideoEnabled: false, // current state - enabled or not
        leavingVideoAcceptableParam: false,
        initializingVideoCall: false,
        isEditingBigText: false,
        fileUploadOverallProgress: 0,
        shouldShowSendMessageButtons: true,
        hasNewMessages: false,
        chatDto: chatDtoFactory(),
        sendMessageAfterMediaInsert: false,
        oppositeUserLastLoginDateTime: null,
        correlationId: null,
        videoTokenId: null,
        compactMessages: false,
        videoPosition: null,
        presenterEnabled: false,
    }
  },
  actions: {
    unsetUser() {
      this.currentUser = null
    },
    fetchUserProfile() {
        return axios.get(`/api/aaa/profile`).then(( {data} ) => {
            console.debug("fetched profile =", data);
            this.currentUser = data;
            bus.emit(PROFILE_SET);

            return axios.get("/api/aaa/settings/init").then(({data}) => {
                const lang = data.language;
                setStoredLanguage(lang);
            })
        });
    },
    fetchAvailableOauth2Providers() {
          return axios.get(`/api/aaa/oauth2/providers`).then(( {data} ) => {
              console.debug("fetched oauth2 providers =", data);
              this.availableOAuth2Providers = data.map(p => p.providerName);
              for (const p of data) {
                  this.OAuth2ProvidersAllowUnbind[p.providerName] = p.allowUnbind;
              }
          });
    },
    updateRedDot() {
        setIcon(this.notificationsCount > 0 || this.hasNewMessages);
    },
    setNotificationCount(count){
      this.notificationsCount = count;
      this.updateRedDot();
    },
    fetchNotificationsCount() {
      axios.get(`/api/notification/count`).then(( {data} ) => {
        console.debug("fetched notifications =", data);
        this.setNotificationCount(data.totalCount);
      });
    },
    fetchHasNewMessages() {
      axios.get(`/api/chat/has-new-messages`).then(( {data} ) => {
          console.debug("fetched has-new-messages =", data);
          this.setHasNewMessages(data.hasUnreadMessages);
      });
    },
    setHasNewMessages(value){
      this.hasNewMessages = value;
      this.updateRedDot();
    },
    unsetNotificationsAndHasNewMessages() {
      this.notificationsCount = 0;
      this.hasNewMessages = false;
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
    resetChatDto() {
      this.chatDto = chatDtoFactory();
    },
    setChatDto(d) {
       this.chatDto = d;
    },
    resetFileUploadingSessionType() {
      this.fileUploadingSessionType = null;
    },
    setFileUploadingSessionType(v) {
      this.fileUploadingSessionType = v;
    },
    resetSendMessageAfterMediaInsertRoutine() {
      this.sendMessageAfterMediaInsert = false;
      this.resetFileUploadingSessionType();
    },
    canDeleteParticipant(userId) {
        return this.chatDto.canEdit && userId != this.currentUser.id
    },
    canVideoKickParticipant(userId) {
        return this.chatDto.canVideoKick && userId != this.currentUser.id
    },
    canAudioMuteParticipant(userId) {
        return this.chatDto.canAudioMute && userId != this.currentUser.id
    },
  },

})
