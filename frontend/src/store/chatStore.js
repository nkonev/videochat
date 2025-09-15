// Utilities
import { defineStore } from 'pinia'
import axios from "axios";
import {chatEditMessageDtoFactory, hasLength, isMobileBrowser, setIcon} from "@/utils";
import {SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import {setStoredLanguage} from "@/store/localStore";
import bus, {NOTIFICATION_COUNT_CHANGED, PROFILE_SET} from "@/bus/bus.js";

export const callStateReady = "ready"
export const callStateInCall = "inCall"
// just a regular upload from MessageEdit's button
export const fileUploadingSessionTypeMessageEdit = "fromMessageEdit"
// when upload embed image, video or recorded video
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
        titleStrike: false,
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
        sendMessageAfterUploadsUploaded: false,
        sendMessageAfterMediaNumFiles: 0,
        sendMessageAfterNumFiles: 0,
        oppositeUserLastSeenDateTime: null,
        correlationId: null,
        videoTokenId: null,
        compactMessages: false,
        videoPosition: null,
        presenterEnabled: false,
        pinnedTrackSid: null,
        oppositeUserInVideo: false,
        videoMiniaturesEnabled: true,
        videoMessagesEnabled: true,
        oppositeUserOnline: false,
        editMessageDto: chatEditMessageDtoFactory(),
        aaaSessionPingInterval: -1, // in milliseconds
        minPasswordLength: 0,

        canShowPinnedLink: true,
        showTempGoTo: false,
        tempGoToChatId: null,
        tempGoToText: null,
        tempGoToMessageId: null,

        presenterUseCover: false,
    }
  },
  actions: {
    unsetUser() {
      this.currentUser = null
    },
    fetchUserProfile() {
        this.incrementProgressCount();
        return axios.get(`/api/aaa/profile`).then(( {data} ) => {
            console.debug("fetched profile =", data);
            this.currentUser = data;
            bus.emit(PROFILE_SET);

            return axios.get("/api/aaa/settings/init").then(({data}) => {
                const lang = data.language;
                setStoredLanguage(lang);
            })
        }).finally(()=>{
            this.decrementProgressCount();
        });
    },
    fetchAaaConfig() {
          return axios.get(`/api/aaa/config`).then(( {data} ) => {
              const providers = data.providers;
              this.availableOAuth2Providers = providers.map(p => p.providerName);
              for (const p of providers) {
                  this.OAuth2ProvidersAllowUnbind[p.providerName] = p.allowUnbind;
              }
              this.aaaSessionPingInterval = data.frontendSessionPingInterval;
              this.minPasswordLength = data.minPasswordLength;
              console.log("config: oauth2 providers", JSON.stringify(this.availableOAuth2Providers), "minPasswordLength", this.minPasswordLength)
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
      return axios.get(`/api/notification/count`).then(( {data} ) => {
        console.debug("fetched notifications =", data);
        this.setNotificationCount(data.totalCount);
        bus.emit(NOTIFICATION_COUNT_CHANGED);
      });
    },
    fetchHasNewMessages() {
      return axios.get(`/api/chat/has-new-messages`).then(( {data} ) => {
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
    cleanFileUploadingQueue() {
      this.fileUploadingQueue = [];
      this.fileUploadOverallProgress = 0;
    },
    fileUploadingQueueHasElements() {
       return !!this.fileUploadingQueue.length
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
      this.sendMessageAfterUploadsUploaded = false;
      this.sendMessageAfterMediaNumFiles = 0;
      this.resetFileUploadingSessionType();
    },
    resetSendMessageAfterFileInsertRoutine() {
      this.sendMessageAfterUploadsUploaded = false;
      this.sendMessageAfterNumFiles = 0;
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
    isMessageEditing() {
        return !!this.editMessageDto.id
    },
    hasMessageEditingText() {
        return hasLength(this.editMessageDto.text)
    },
  },

})
