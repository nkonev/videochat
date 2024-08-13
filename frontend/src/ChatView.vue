<template>
    <splitpanes ref="splOuter" class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar" @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">
      <pane :size="leftPaneSize()" v-if="showLeftPane()">
        <ChatList :embedded="true" v-if="isAllowedChatList()" ref="chatListRef"/>
      </pane>

      <pane style="background: white" :size="centralPaneSize()">
        <splitpanes ref="splCentral" class="default-theme" :dbl-click-splitter="false" horizontal @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">
          <pane v-if="showTopPane()" :size="topPaneSize()">
            <ChatVideo v-if="chatDtoIsReady" :chatId="chatId" ref="chatVideoRef"/>
          </pane>

          <pane :class="messageListPaneClass()" :size="messageListPaneSize()">
            <v-tooltip
                v-if="broadcastMessage"
                :model-value="showTooltip"
                activator=".message-edit-pane"
                location="bottom start"
            >
              <span v-html="broadcastMessage"></span>
            </v-tooltip>

            <div v-if="pinnedPromoted" :key="pinnedPromotedKey" class="pinned-promoted" :title="$vuetify.locale.t('$vuetify.pinned_message')">
              <v-alert
                  color="red-lighten-4"
                  elevation="2"
                  density="compact"
              >
                <router-link :to="getPinnedRouteObject(pinnedPromoted)" class="pinned-text" v-html="pinnedPromoted.text"></router-link>
              </v-alert>
            </div>

            <MessageList :canResend="chatStore.chatDto.canResend" :blog="chatStore.chatDto.blog" :isCompact="isVideoRoute()"/>

            <v-btn v-if="chatStore.showScrollDown" variant="elevated" color="primary" icon="mdi-arrow-down-thick" :class="scrollDownClass()" @click="scrollDown()"></v-btn>
            <v-btn v-if="isMobile()" variant="elevated" color="primary" icon="mdi-plus" class="new-fab-b" @click="openNewMessageDialog()"></v-btn>
          </pane>
          <pane class="message-edit-pane" v-if="showBottomPane()" :size="bottomPaneSize()">
            <MessageEdit :chatId="this.chatId"/>
          </pane>
        </splitpanes>
      </pane>

      <pane v-if="showRightPane()" :size="rightPaneSize()">
        <ChatVideo v-if="chatDtoIsReady" :chatId="chatId" ref="chatVideoRef"/>
      </pane>

    </splitpanes>
</template>

<script>
import { Splitpanes, Pane } from 'splitpanes';
import ChatList from "@/ChatList.vue";
import MessageList from "@/MessageList.vue";
import MessageEdit from "@/MessageEdit.vue";
import heightMixin from "@/mixins/heightMixin";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import axios from "axios";
import {
    hasLength,
    isCalling,
    isChatRoute,
    new_message,
    setTitle,
    goToPreservingQuery
} from "@/utils";
import bus, {
    CHAT_DELETED,
    CHAT_EDITED,
    FILE_CREATED,
    FILE_REMOVED,
    FILE_UPDATED,
    LOGGED_OUT,
    MESSAGE_ADD,
    MESSAGE_BROADCAST,
    MESSAGE_DELETED,
    MESSAGE_EDITED, MESSAGES_RELOAD,
    OPEN_EDIT_MESSAGE,
    PARTICIPANT_ADDED,
    PARTICIPANT_DELETED,
    PARTICIPANT_EDITED, PINNED_MESSAGE_EDITED,
    PINNED_MESSAGE_PROMOTED,
    PINNED_MESSAGE_UNPROMOTED,
    PREVIEW_CREATED,
    PROFILE_SET,
    PUBLISHED_MESSAGE_ADD, PUBLISHED_MESSAGE_EDITED,
    PUBLISHED_MESSAGE_REMOVE,
    REACTION_CHANGED,
    REACTION_REMOVED,
    REFRESH_ON_WEBSOCKET_RESTORED,
    SCROLL_DOWN,
    USER_TYPING,
    VIDEO_CALL_USER_COUNT_CHANGED,
    VIDEO_DIAL_STATUS_CHANGED
} from "@/bus/bus";
import {chat_list_name, chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import ChatVideo from "@/ChatVideo.vue";
import videoPositionMixin from "@/mixins/videoPositionMixin";
import {SEARCH_MODE_CHATS, searchString} from "@/mixins/searchString.js";
import onFocusMixin from "@/mixins/onFocusMixin.js";

const getChatEventsData = (message) => {
  return message.data?.chatEvents
};

let writingUsersTimerId;

const panelSizesKey = "panelSizes";

const emptyStoredPanes = () => {
  return {
    topPane: 60, // ChatVideo mobile
    leftPane: 20, // ChatList
    rightPane: 40, // ChatVideo desktop
    bottomPane: 20, // MessageEdit in case desktop (!isMobile())
    bottomPaneBig: 60, // MessageEdit in case desktop (!isMobile()) and a text containing a newline
  }
}

export default {
  mixins: [
    heightMixin(),
    videoPositionMixin(),
    searchString(SEARCH_MODE_CHATS),
    onFocusMixin(),
  ],
  data() {
    return {
      pinnedPromoted: null,
      pinnedPromotedKey: +new Date(),
      writingUsers: [],
      showTooltip: true,
      broadcastMessage: null,
      // shows that all the possible PUT /join have happened and we can get ChatList. Intentionally doesn't reset on switching chat at left
      // if we remove it (or replace with chatDtoIsReady) - there are going to be disappears of ChatList when user clicks on the different chat
      initialLoaded: false,
      chatEventsSubscription: null,
    }
  },
  components: {
    ChatVideo,
    Splitpanes,
    Pane,
    ChatList,
    MessageList,
    MessageEdit,
  },
  computed: {
    ...mapStores(useChatStore),
    chatId() {
      return this.$route.params.id
    },
    chatDtoIsReady() {
      return !!this.chatStore.chatDto.id
    },
  },
  methods: {
    onProfileSet() {
      return this.getInfo(this.chatId).then(()=>{
        this.chatStore.showCallManagement = true;
        this.chatEventsSubscription.graphQlSubscribe();
      })
    },
    onLogout() {
      this.partialReset();
      this.initialLoaded = false;
      this.chatEventsSubscription.graphQlUnsubscribe();
    },
    fetchAndSetChat(chatId) {
      return axios.get(`/api/chat/${chatId}`, {
        signal: this.requestAbortController.signal
      }).then((response) => {
        if (response.status == 205) {
          return axios.put(`/api/chat/${chatId}/join`, null, {
            signal: this.requestAbortController.signal
          }).then((response)=>{
              return axios.get(`/api/chat/${chatId}`, {
                signal: this.requestAbortController.signal
              }).then((response)=>{
                  return this.processNormalInfoResponse(response)
              })
          })
        } else if (response.status == 204) {
          this.goToChatList();
          this.setWarning(this.$vuetify.locale.t('$vuetify.chat_not_found'));
          return Promise.reject();
        } else {
            return this.processNormalInfoResponse(response);
        }
      })
    },
    processNormalInfoResponse(response) {
        const data = response.data;
        console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
        this.commonChatEdit(data);
        this.chatStore.tetATet = data.tetATet;
        this.chatStore.setChatDto(data);
        this.initialLoaded = true;
        return Promise.resolve();
    },
    commonChatEdit(data) {
        this.chatStore.title = data.name;
        setTitle(data.name);
        this.chatStore.avatar = data.avatar;
        this.chatStore.chatUsersCount = data.participantsCount;
        this.chatStore.showChatEditButton = data.canEdit;
        this.chatStore.canBroadcastTextMessage = data.canBroadcast;
        if (data.blog) {
            this.chatStore.showGoToBlogButton = this.chatId;
        } else {
            this.chatStore.showGoToBlogButton = null;
        }
        if (data.tetATet && data.participantsCount == 2) {
            this.chatStore.oppositeUserLastLoginDateTime = data.lastLoginDateTime;
        }
    },
    fetchPromotedMessage(chatId) {
      axios.get(`/api/chat/${chatId}/message/pin/promoted`, {
        signal: this.requestAbortController.signal
      }).then((response) => {
        if (response.status != 204) {
          this.pinnedPromoted = response.data;
          this.pinnedPromotedKey++
        } else {
          this.pinnedPromoted = null;
        }
      });
    },
    getInfo(chatId) {
      return this.fetchAndSetChat(chatId).then(() => {
        // async call
        this.fetchPromotedMessage(chatId);
        axios.get(`/api/video/${chatId}/users`, {
          signal: this.requestAbortController.signal
        })
          .then(response => response.data)
          .then(data => {
            bus.emit(VIDEO_CALL_USER_COUNT_CHANGED, data);
            this.chatStore.videoChatUsersCount = data.usersCount;
          })
        return Promise.resolve();
      }).then(() => {
        // async call
        axios.get(`/api/video/${chatId}/record/status`, {
          signal: this.requestAbortController.signal
        }).then(({data}) => {
          this.chatStore.canMakeRecord = data.canMakeRecord;
          if (data.canMakeRecord) {
            const record = data.recordInProcess;
            if (record) {
              this.chatStore.showRecordStopButton = true;
            }
          }
        })
        return Promise.resolve();
      })
    },
    goToChatList() {
      this.$router.push(({name: chat_list_name}))
    },
    getGraphQlSubscriptionQuery() {
      return `
                                fragment DisplayMessageDtoFragment on DisplayMessageDto {
                                  id
                                  text
                                  chatId
                                  ownerId
                                  createDateTime
                                  editDateTime
                                  owner {
                                    id
                                    login
                                    avatar
                                    shortInfo
                                    loginColor
                                  }
                                  canEdit
                                  canDelete
                                  fileItemUuid
                                  embedMessage {
                                    id
                                    chatId
                                    chatName
                                    text
                                    owner {
                                      id
                                      login
                                      avatar
                                      shortInfo
                                      loginColor
                                    }
                                    embedType
                                    isParticipant
                                  }
                                  pinned
                                  blogPost
                                  pinnedPromoted
                                  reactions {
                                    count
                                    users {
                                      id
                                      login
                                      avatar
                                      shortInfo
                                      loginColor
                                    }
                                    reaction
                                  }
                                  published
                                  canPublish
                                  canPin
                                }

                                subscription{
                                  chatEvents(chatId: ${this.chatId}) {
                                    eventType
                                    messageEvent {
                                      ...DisplayMessageDtoFragment
                                    }
                                    messageDeletedEvent {
                                      id
                                      chatId
                                    }
                                    userTypingEvent {
                                      login
                                      participantId
                                    }
                                    messageBroadcastEvent {
                                      login
                                      userId
                                      text
                                    }
                                    previewCreatedEvent {
                                      id
                                      url
                                      previewUrl
                                      aType
                                      correlationId
                                    }
                                    participantsEvent {
                                      id
                                      login
                                      avatar
                                      admin
                                      shortInfo
                                      loginColor
                                    }
                                    promoteMessageEvent {
                                      count
                                      message {
                                        id
                                        text
                                        chatId
                                        ownerId
                                        owner {
                                          id
                                          login
                                          avatar
                                          shortInfo
                                          loginColor
                                        }
                                        pinnedPromoted
                                        createDateTime
                                        canPin
                                      }
                                    }
                                    publishedMessageEvent {
                                      count
                                      message {
                                        id
                                        text
                                        chatId
                                        ownerId
                                        owner {
                                          id
                                          login
                                          avatar
                                          shortInfo
                                          loginColor
                                        }
                                        createDateTime
                                        canPublish
                                      }
                                    }
                                    fileEvent {
                                      fileInfoDto {
                                        id
                                        filename
                                        url
                                        publicUrl
                                        previewUrl
                                        size
                                        canDelete
                                        canEdit
                                        canShare
                                        lastModified
                                        ownerId
                                        owner {
                                          id
                                          login
                                          avatar
                                        }
                                        canPlayAsVideo
                                        canShowAsImage
                                        canPlayAsAudio
                                        fileItemUuid
                                        correlationId
                                      }
                                    }
                                    reactionChangedEvent {
                                      messageId
                                      reaction {
                                        count
                                        users {
                                          id
                                          login
                                          avatar
                                          shortInfo
                                          loginColor
                                        }
                                        reaction
                                      }
                                    }
                                  }
                                }
                `
    },
    onNextSubscriptionElement(e) {
      if (getChatEventsData(e).eventType === 'message_created') {
        const d = getChatEventsData(e).messageEvent;
        bus.emit(MESSAGE_ADD, d);
      } else if (getChatEventsData(e).eventType === 'message_deleted') {
        const d = getChatEventsData(e).messageDeletedEvent;
        bus.emit(MESSAGE_DELETED, d);
      } else if (getChatEventsData(e).eventType === 'message_edited') {
        const d = getChatEventsData(e).messageEvent;
        bus.emit(MESSAGE_EDITED, d);
      } else if (getChatEventsData(e).eventType === "user_typing") {
        const d = getChatEventsData(e).userTypingEvent;
        bus.emit(USER_TYPING, d);
      } else if (getChatEventsData(e).eventType === "user_broadcast") {
        const d = getChatEventsData(e).messageBroadcastEvent;
        bus.emit(MESSAGE_BROADCAST, d);
      } else if (getChatEventsData(e).eventType === "preview_created") {
        const d = getChatEventsData(e).previewCreatedEvent;
        bus.emit(PREVIEW_CREATED, d);
      } else if (getChatEventsData(e).eventType === "participant_added") {
        const d = getChatEventsData(e).participantsEvent;
        bus.emit(PARTICIPANT_ADDED, d);
      } else if (getChatEventsData(e).eventType === "participant_deleted") {
        const d = getChatEventsData(e).participantsEvent;
        bus.emit(PARTICIPANT_DELETED, d);
      } else if (getChatEventsData(e).eventType === "participant_edited") {
        const d = getChatEventsData(e).participantsEvent;
        bus.emit(PARTICIPANT_EDITED, d);
      } else if (getChatEventsData(e).eventType === "pinned_message_promote") {
        const d = getChatEventsData(e).promoteMessageEvent;
        bus.emit(PINNED_MESSAGE_PROMOTED, d);
      } else if (getChatEventsData(e).eventType === "pinned_message_unpromote") {
        const d = getChatEventsData(e).promoteMessageEvent;
        bus.emit(PINNED_MESSAGE_UNPROMOTED, d);
      } else if (getChatEventsData(e).eventType === "pinned_message_edit") {
        const d = getChatEventsData(e).promoteMessageEvent;
        bus.emit(PINNED_MESSAGE_EDITED, d);
      } else if (getChatEventsData(e).eventType === "published_message_add") {
          const d = getChatEventsData(e).publishedMessageEvent;
          bus.emit(PUBLISHED_MESSAGE_ADD, d);
      } else if (getChatEventsData(e).eventType === "published_message_remove") {
          const d = getChatEventsData(e).publishedMessageEvent;
          bus.emit(PUBLISHED_MESSAGE_REMOVE, d);
      } else if (getChatEventsData(e).eventType === "published_message_edit") {
          const d = getChatEventsData(e).publishedMessageEvent;
          bus.emit(PUBLISHED_MESSAGE_EDITED, d);
      } else if (getChatEventsData(e).eventType === "file_created") {
        const d = getChatEventsData(e).fileEvent;
        bus.emit(FILE_CREATED, d);
      } else if (getChatEventsData(e).eventType === "file_removed") {
        const d = getChatEventsData(e).fileEvent;
        bus.emit(FILE_REMOVED, d);
      } else if (getChatEventsData(e).eventType === "file_updated") {
        const d = getChatEventsData(e).fileEvent;
        bus.emit(FILE_UPDATED, d);
      } else if (getChatEventsData(e).eventType === "reaction_changed") {
        const d = getChatEventsData(e).reactionChangedEvent;
        bus.emit(REACTION_CHANGED, d);
      } else if (getChatEventsData(e).eventType === "reaction_removed") {
        const d = getChatEventsData(e).reactionChangedEvent;
        bus.emit(REACTION_REMOVED, d);
      } else if (getChatEventsData(e).eventType === "messages_reload") {
          bus.emit(MESSAGES_RELOAD);
      }
    },
    getPinnedRouteObject(item) {
      const routeName = this.isVideoRoute() ? videochat_name : chat_name;
      return {name: routeName, params: {id: item.chatId}, hash: messageIdHashPrefix + item.id};
    },
    onPinnedMessagePromoted(item) {
      this.pinnedPromoted = item.message;
      this.pinnedPromotedKey++;
    },
    onPinnedMessageUnpromoted(item) {
      if (this.pinnedPromoted && this.pinnedPromoted.id == item.message.id) {
        this.pinnedPromoted = null;
      }
    },
    onPinnedMessageChanged(item) {
        if (this.pinnedPromoted && this.pinnedPromoted.id == item.message.id) {
            this.onPinnedMessagePromoted(item);
        }
    },
    onFocus() {
        if (this.chatStore.currentUser && this.chatId) {
            this.getInfo(this.chatId);
        }
    },
    onUserTyping(data) {
      console.debug("OnUserTyping", data);

      if (this.chatStore.currentUser?.id == data.participantId) {
        console.log("Skipping myself typing notifications");
        return;
      }
      this.showTooltip = true;

      const idx = this.writingUsers.findIndex(value => value.login === data.login);
      if (idx !== -1) {
        this.writingUsers[idx].timestamp = + new Date();
      } else {
        this.writingUsers.push({timestamp: +new Date(), login: data.login})
      }

      this.chatStore.usersWritingSubtitleInfo = this.writingUsers.map(v=>v.login).join(', ') + " " + this.$vuetify.locale.t('$vuetify.user_is_writing');
    },
    onUserBroadcast(dto) {
      console.log("onUserBroadcast", dto);
      const stripped = dto.text;
      if (stripped && stripped.length > 0) {
        this.showTooltip = true;
        this.broadcastMessage = dto.text;
      } else {
        this.broadcastMessage = null;
      }
    },
    onChatChange(data) {
        if (data.id == this.chatId) {
            this.commonChatEdit(data);
            this.chatStore.setChatDto(data);
        }
    },
    onChatDelete(dto) {
      if (dto.id == this.chatId) {
          this.$router.push(({name: chat_list_name}))
      }
    },
    onParticipantDeleted(dtos) { // also there is redraw/delete logic in ChatList::redrawItem
        if (dtos.find(p => p.id == this.chatStore.currentUser.id)) {
            const routerNewState = { name: chat_list_name};
            goToPreservingQuery(this.$route, this.$router, routerNewState);
        }
    },
    isAllowedVideo() {
      return this.chatStore.currentUser && this.$route.name == videochat_name && this.chatStore.chatDto?.participantIds?.length
    },
    isAllowedChatList() {
        // second condition is for waiting full loading (including PUT /join) of chat in order to be visible
        // testcase: user 1 creates blog without any other users
        // user 2 wants to write a comment, clicking the button in blog
        // he is being redirected to chat, because user 2 is not a participant, he gets a http code and the browser issues /join
        // after the joining all user 2 want to see chat of blog at the left
        return this.chatStore.currentUser && this.initialLoaded
    },
    onVideoCallChanged(dto) {
      if (dto.chatId == this.chatId) {
        this.chatStore.videoChatUsersCount = dto.usersCount;
      }
    },
    onWsRestoredRefresh() {
      this.getInfo(this.chatId)
    },
    partialReset() {
      this.chatStore.resetChatDto();

      this.chatStore.videoChatUsersCount=0;
      this.chatStore.canMakeRecord=false;
      this.pinnedPromoted=null;
      this.chatStore.canBroadcastTextMessage = false;
      this.chatStore.showRecordStartButton = false;
      this.chatStore.showRecordStopButton = false;
      this.chatStore.showChatEditButton = false;

      this.chatStore.title = null;
      setTitle(null);
      this.chatStore.avatar = null;
      this.chatStore.showGoToBlogButton = null;

      this.chatStore.chatUsersCount = 0;

      this.chatStore.oppositeUserLastLoginDateTime = null;

      this.chatStore.showCallManagement = false;
    },
    onChatDialStatusChange(dto) {
      if (this.chatStore.chatDto?.tetATet && dto.chatId == this.chatId) { // if tet-a-tet
        for (const videoDialChanged of dto.dials) {
          if (this.chatStore.currentUser.id != videoDialChanged.userId) { // if counterpart exists
            this.chatStore.shouldPhoneBlink = isCalling(videoDialChanged.status); // and if their status is being calling - turn on blinking on my frontend
          }
        }
      }
    },
    openNewMessageDialog() { // on mobile OPEN_EDIT_MESSAGE with the null argument
      bus.emit(OPEN_EDIT_MESSAGE, {dto: null, actionType: new_message});
    },
    messageListPaneClass() {
      const classes = [];
      classes.push('message-pane');
      if (this.isMobile()) {
        classes.push('message-pane-mobile');
      }
      return classes;
    },

    showRightPane() {
      return !this.isMobile() && this.isAllowedVideo()
    },

    showLeftPane() {
      return this.shouldShowChatList()
    },
    showBottomPane() {
      return !this.isMobile();
    },

    leftPaneSize() {
      return this.getStored().leftPane;
    },
    rightPaneSize() {
      return this.getStored().rightPane;
    },
    showTopPane() {
      return this.isMobile() && this.isAllowedVideo()
    },
    topPaneSize() {
      return this.getStored().topPane;
    },
    centralPaneSize() {
      if (this.isMobile()) {
        return 100
      } else {
        if (this.showRightPane()) {
          return 100 - this.rightPaneSize();
        } else if (this.showLeftPane()) {
          return 100 - this.leftPaneSize();
        } else {
          return 100;
        }
      }
    },
    bottomPaneSize() {
      if (!this.chatStore.isEditingBigText) {
        return this.getStored().bottomPane;
      } else {
        return this.getStored().bottomPaneBig;
      }
    },
    messageListPaneSize() {
      if (this.isMobile()) {
        if (this.showTopPane()) {
          return 100 - this.topPaneSize();
        } else {
          return 100;
        }
      } else {
        if (this.showBottomPane()) {
          return 100 - this.bottomPaneSize()
        } else {
          return 100
        }
      }
    },

    // returns json with sizes from localstore
    getStored() {
      const mbItem = localStorage.getItem(panelSizesKey);
      if (!mbItem) {
        return emptyStoredPanes();
      } else {
        return JSON.parse(mbItem);
      }
    },
    // saves to localstore
    saveToStored(obj) {
      localStorage.setItem(panelSizesKey, JSON.stringify(obj));
    },
    // prepares json to store by extracting concrete panel sizes
    prepareForStore() {
      const outerPaneSizes = this.$refs.splOuter.panes.map(i => i.size);
      const centralPaneSizes = this.$refs.splCentral.panes.map(i => i.size);
      const ret = this.getStored();
      if (this.isMobile()) {
        if (this.showTopPane()) {
          const topPaneSize = centralPaneSizes[0];
          ret.topPane = topPaneSize;
        }
      } else {
        if (this.showLeftPane()) {
          ret.leftPane = outerPaneSizes[0];
        }
        if (this.showRightPane()) {
          ret.rightPane = outerPaneSizes[outerPaneSizes.length - 1]
        }
        if (this.showBottomPane()) {
          const bottomPaneSize = centralPaneSizes[centralPaneSizes.length - 1];
          if (!this.chatStore.isEditingBigText) {
            ret.bottomPane = bottomPaneSize;
          } else {
            ret.bottomPaneBig = bottomPaneSize;
          }
        }
      }
      // console.debug("Preparing for store", ret)
      return ret
    },
    // sets concrete panel sizes
    restorePanelsSize(ret) {
      // console.debug("Restoring from", ret);
      if (this.isMobile()) {
        if (this.showTopPane()) {
          this.$refs.splCentral.panes[0].size = ret.topPane;
        }
      } else {
        if (this.showLeftPane()) {
          this.$refs.splOuter.panes[0].size = ret.leftPane;
        }
        if (this.showRightPane()) {
          this.$refs.splOuter.panes[this.$refs.splOuter.panes.length - 1].size = ret.rightPane;
        }
        if (this.showBottomPane()) {
          let bottomPaneSize;
          if (!this.chatStore.isEditingBigText) {
            bottomPaneSize = ret.bottomPane;
          } else {
            bottomPaneSize = ret.bottomPaneBig;
          }
          this.$refs.splCentral.panes[this.$refs.splCentral.panes.length - 1].size = bottomPaneSize;
        }
      }
    },

    onPanelAdd() {
      this.$refs.chatVideoRef?.recalculateLayout();

      // console.debug("On panel add", this.$refs.splOuter.panes);
      this.$nextTick(() => {
        const stored = this.getStored();
        // console.debug("Restoring on add", stored)
        this.restorePanelsSize(stored);
      })

    },
    onPanelRemove() {
      this.$refs.chatVideoRef?.recalculateLayout();

      // console.debug("On panel removed", this.$refs.splOuter.panes);
      this.$nextTick(() => {
        const stored = this.getStored();
        // console.debug("Restoring on remove", stored)
        this.restorePanelsSize(stored);
      })
    },
    onPanelResized() {
      this.$refs.chatVideoRef?.recalculateLayout();

      this.$nextTick(() => {
        this.saveToStored(this.prepareForStore());
      })
    },
    scrollDown () {
      bus.emit(SCROLL_DOWN)
    },
    scrollDownClass() {
      if (this.isMobile()) {
        return "new-fab-t"
      } else {
        return "new-fab-b"
      }
    },
  },
  watch: {
    '$route': {
      handler: function (newValue, oldValue) {
        if (isChatRoute(newValue)) {
          if (newValue.params.id != oldValue.params.id) {
            console.debug("Chat id has been changed", oldValue.params.id, "->", newValue.params.id);
            if (hasLength(newValue.params.id)) {
              this.chatStore.incrementProgressCount();

              // used for
              // 1. to prevent opening ChatVideo with old (previous) chatDto that contains old chatId
              // 2. to prevent rendering MessageList and get 401
              this.partialReset();
              this.onProfileSet().then(()=>{
                this.chatStore.decrementProgressCount();
              })
            }
          }
        }
      }
    },
  },
  created() {

  },
  async mounted() {
    this.chatStore.title = `Chat #${this.chatId}`;
    this.chatStore.chatUsersCount = 0;
    this.chatStore.isShowSearch = true;
    this.chatStore.showChatEditButton = false;

    // create subscription object before ON_PROFILE_SET
    this.chatEventsSubscription = graphqlSubscriptionMixin('chatEvents', this.getGraphQlSubscriptionQuery, this.setErrorSilent, this.onNextSubscriptionElement);

    if (this.chatStore.currentUser) {
      await this.onProfileSet();
    }

    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLogout);
    bus.on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
    bus.on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    bus.on(PINNED_MESSAGE_EDITED, this.onPinnedMessageChanged);
    bus.on(USER_TYPING, this.onUserTyping);
    bus.on(MESSAGE_BROADCAST, this.onUserBroadcast);
    bus.on(CHAT_EDITED, this.onChatChange);
    bus.on(CHAT_DELETED, this.onChatDelete);
    bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
    bus.on(PARTICIPANT_DELETED, this.onParticipantDeleted);

    writingUsersTimerId = setInterval(()=>{
      const curr = + new Date();
      this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
      if (this.writingUsers.length == 0) {
        this.chatStore.usersWritingSubtitleInfo = null;
      }
    }, 500);

    this.installOnFocus();
  },
  beforeUnmount() {
    this.uninstallOnFocus();

    this.chatEventsSubscription.graphQlUnsubscribe();
    this.chatEventsSubscription = null;

    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLogout);
    bus.off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
    bus.off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    bus.off(PINNED_MESSAGE_EDITED, this.onPinnedMessageChanged);
    bus.off(USER_TYPING, this.onUserTyping);
    bus.off(MESSAGE_BROADCAST, this.onUserBroadcast);
    bus.off(CHAT_EDITED, this.onChatChange);
    bus.off(CHAT_DELETED, this.onChatDelete);
    bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
    bus.off(PARTICIPANT_DELETED, this.onParticipantDeleted);

    this.chatStore.isShowSearch = false;

    this.partialReset();
    clearInterval(writingUsersTimerId);
    this.initialLoaded = false;

    this.chatStore.isEditingBigText = false;
  }
}
</script>


<style scoped lang="stylus">
@import "pinned.styl"

.pinned-promoted {
  position: absolute
  left 0
  margin-right 2em
  z-index: 4;
}
.message-pane {
  position: relative // needed for the correct displaying .pinned-promoted
}
.message-pane-mobile {
    align-items: unset;
}

.new-fab-b {
  position: absolute
  bottom: 20px
  right: 20px
  z-index: 1000
}

.new-fab-t {
  position: absolute
  bottom: 90px
  right: 20px
  z-index: 1000
}

@media screen and (max-width: $mobileWidth) {
    .pinned-promoted {
        margin-right unset
    }
}
</style>

<style lang="stylus">
.pinned-promoted {
  .v-alert__content{
    text-overflow: ellipsis;
  }
  .v-alert {
    padding-top 2px
    padding-bottom 2px
  }

}
</style>
