<template>
    <splitpanes ref="splOuter" class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar" @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">
      <pane :size="leftPaneSize()" v-if="showLeftPane()">
        <ChatList :embedded="true" v-if="isAllowedChatList()"/>
      </pane>

      <pane>
        <splitpanes ref="splInner" class="default-theme" :dbl-click-splitter="false" horizontal @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">
          <pane v-if="showTopPane()" min-size="15" :size="topPaneSize()">
            <ChatVideo :chatDto="chatDto" :videoIsOnTopProperty="videoIsOnTop()" />
          </pane>

          <pane style="width: 100%; background: white" :class="messageListPaneClass()">
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
                  closable
                  color="red-lighten-4"
                  elevation="2"
                >
                  <router-link :to="getPinnedRouteObject(pinnedPromoted)" class="pinned-text" v-html="pinnedPromoted.text">
                  </router-link>
                </v-alert>
              </div>

              <MessageList v-if="isAllowedMessageList()" :chatDto="chatDto"/>

              <v-btn v-if="isMobile()" variant="elevated" color="primary" icon="mdi-plus" class="new-fab" @click="openNewMessageDialog()"></v-btn>
          </pane>
          <pane class="message-edit-pane" v-if="showBottomPane()" :size="bottomPaneSize()">
            <MessageEdit :chatId="this.chatId"/>
          </pane>
        </splitpanes>
      </pane>
      <pane v-if="showRightPane()" min-size="15" :size="rightPaneSize()">
        <ChatVideo :chatDto="chatDto" :videoIsOnTop="videoIsOnTop()"/>
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
import {hasLength, isCalling, isChatRoute, new_message, setTitle} from "@/utils";
import bus, {
  CHAT_DELETED,
  CHAT_EDITED,
  FILE_CREATED, FILE_REMOVED, FILE_UPDATED, FOCUS, LOGGED_OUT,
  MESSAGE_ADD,
  MESSAGE_BROADCAST,
  MESSAGE_DELETED,
  MESSAGE_EDITED, OPEN_EDIT_MESSAGE,
  PARTICIPANT_ADDED,
  PARTICIPANT_DELETED,
  PARTICIPANT_EDITED,
  PINNED_MESSAGE_PROMOTED,
  PINNED_MESSAGE_UNPROMOTED,
  PREVIEW_CREATED,
  PROFILE_SET, REACTION_CHANGED, REACTION_REMOVED, REFRESH_ON_WEBSOCKET_RESTORED,
  USER_TYPING,
  VIDEO_CALL_USER_COUNT_CHANGED, VIDEO_DIAL_STATUS_CHANGED
} from "@/bus/bus";
import {chat_list_name, chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import ChatVideo from "@/ChatVideo.vue";
import videoPositionMixin from "@/mixins/videoPositionMixin";

const getChatEventsData = (message) => {
  return message.data?.chatEvents
};

let writingUsersTimerId;

const MESSAGE_EDIT_PANE_SIZE_INITIAL = 20; // percents
const MESSAGE_EDIT_PANE_SIZE_EXPANDED = 60; // percents
const ONTOP_VIDEO_PANE_SIZE = 30;

const panelSizesKey = "panelSizes";

const emptyStoredPanes = () => {
  return {
    leftPane: 25, // ChatList
    rightPane: 40, // ChatVideo in case videoIsAtSide()
    topPane: 30, // ChatVideo in case videoIsOnTop()
    bottomPane: 20, // MessageEdit in case desktop (!isMobile())
    bottomPaneBig: 60, // MessageEdit in case desktop (!isMobile()) and a text containing a newline
  }
}

export default {
  mixins: [
    heightMixin(),
    graphqlSubscriptionMixin('chatEvents'),
    videoPositionMixin(),
  ],
  data() {
    return {
      chatDto: null,
      pinnedPromoted: null,
      pinnedPromotedKey: +new Date(),
      writingUsers: [],
      showTooltip: true,
      broadcastMessage: null,
      isSwitching: false,
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
  },
  methods: {
    onProfileSet() {
      return this.getInfo().then(()=>{
        this.graphQlSubscribe();
      })
    },
    onLogout() {
      this.partialReset();
      this.graphQlUnsubscribe();
    },
    fetchAndSetChat() {
      return axios.get(`/api/chat/${this.chatId}`).then((response) => {
        if (response.status == 204) {
          this.goToChatList();
          this.setWarning(this.$vuetify.locale.t('$vuetify.chat_not_found'));
          return Promise.reject();
        } else {
          const data = response.data;
          console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
          this.commonChatEdit(data);
          this.chatStore.tetATet = data.tetATet;
          this.chatDto = data;
          return Promise.resolve();
        }
      })
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
    },
    fetchPromotedMessage() {
      axios.get(`/api/chat/${this.chatId}/message/pin/promoted`).then((response) => {
        if (response.status != 204) {
          this.pinnedPromoted = response.data;
          this.pinnedPromotedKey++
        } else {
          this.pinnedPromoted = null;
        }
      });
    },
    getInfo() {
      return this.fetchAndSetChat().then(() => {
        // async call
        this.fetchPromotedMessage();
        axios.get(`/api/video/${this.chatId}/users`)
          .then(response => response.data)
          .then(data => {
            bus.emit(VIDEO_CALL_USER_COUNT_CHANGED, data);
            this.chatStore.videoChatUsersCount = data.usersCount;
          })
        return Promise.resolve();
      }).then(() => {
        // async call
        axios.get(`/api/video/${this.chatId}/record/status`).then(({data}) => {
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
                                    }
                                    embedType
                                    isParticipant
                                  }
                                  pinned
                                  blogPost
                                  pinnedPromoted
                                  reactions {
                                    count
                                    reaction
                                  }
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
                                    }
                                    promoteMessageEvent {
                                      totalCount
                                      message {
                                        ...DisplayMessageDtoFragment
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
                                      }
                                      count
                                    }
                                    reactionChangedEvent {
                                      messageId
                                      reaction {
                                        count
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
    onFocus() {
        if (this.chatStore.currentUser) {
            this.getInfo();
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

      this.chatStore.moreImportantSubtitleInfo = this.writingUsers.map(v=>v.login).join(', ') + " " + this.$vuetify.locale.t('$vuetify.user_is_writing');
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
            this.chatDto = data;
        }
    },
    onChatDelete(dto) {
      if (dto.id == this.chatId) {
          this.$router.push(({name: chat_list_name}))
      }
    },
    isAllowedVideo() {
      return this.chatStore.currentUser && this.$route.name == videochat_name && !this.isSwitching && this.chatDto?.participantIds?.length
    },
    isAllowedMessageList() {
      return this.chatStore.currentUser && !!this.chatDto?.id
    },
    isAllowedChatList() {
      return this.chatStore.currentUser && !!this.chatDto?.id
    },
    onVideoCallChanged(dto) {
      if (dto.chatId == this.chatId) {
        this.chatStore.videoChatUsersCount = dto.usersCount;
      }
    },
    onWsRestoredRefresh() {
      this.getInfo()
    },
    partialReset() {
      this.chatDto = null;

      this.chatStore.videoChatUsersCount=0;
      this.chatStore.canMakeRecord=false;
      this.pinnedPromoted=null;
      this.chatStore.canBroadcastTextMessage = false;
      this.chatStore.showRecordStartButton = false;
      this.chatStore.showRecordStopButton = false;
      this.chatStore.showChatEditButton = false;
    },
    onChatDialStatusChange(dto) {
      if (this.chatDto?.tetATet && dto.chatId == this.chatId) {
        for (const videoDialChanged of dto.dials) {
          if (this.chatStore.currentUser.id != videoDialChanged.userId) {
            this.chatStore.shouldPhoneBlink = isCalling(videoDialChanged.status);
          }
        }
      }
    },
    openNewMessageDialog() { // on mobile OPEN_EDIT_MESSAGE with the null argument
      bus.emit(OPEN_EDIT_MESSAGE, {dto: null, actionType: new_message});
    },
    shouldShowVideoOnTop() {
        return this.videoIsOnTop() && this.isAllowedVideo()
    },
    messageListPaneClass() {
      const classes = [];
      classes.push('message-pane');
      if (this.isMobile()) {
        classes.push('message-pane-mobile');
      }
      return classes;
    },


    showLeftPane() {
      return this.shouldShowChatList()
    },
    showRightPane() {
      return this.videoIsAtSide() && this.isAllowedVideo();
    },
    showTopPane() {
      return this.shouldShowVideoOnTop();
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
    topPaneSize() {
      return this.getStored().topPane;
    },
    bottomPaneSize() {
      if (!this.chatStore.isEditingBigText) {
        return this.getStored().bottomPane;
      } else {
        return this.getStored().bottomPaneBig;
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
      const innerPaneSizes = this.$refs.splInner.panes.map(i => i.size);
      const ret = this.getStored();
      if (this.showLeftPane()) {
        ret.leftPane = outerPaneSizes[0];
      }
      if (this.showRightPane()) {
        ret.rightPane = outerPaneSizes[outerPaneSizes.length - 1]
      }
      if (this.showTopPane()) {
        ret.topPane = innerPaneSizes[0]
      }
      if (this.showBottomPane()) {
        const bottomPaneSize = innerPaneSizes[innerPaneSizes.length - 1];
        if (!this.chatStore.isEditingBigText) {
          ret.bottomPane = bottomPaneSize;
        } else {
          ret.bottomPaneBig = bottomPaneSize;
        }
      }
      // console.debug("Preparing for store", ret)
      return ret
    },
    // sets concrete panel sizes
    restorePanelsSize(ret) {
      // console.debug("Restoring from", ret);
      if (this.showLeftPane()) {
        this.$refs.splOuter.panes[0].size = ret.leftPane;
      }
      if (this.showRightPane()) {
        this.$refs.splOuter.panes[this.$refs.splOuter.panes.length - 1].size = ret.rightPane;
      }
      if (this.showTopPane()) {
        this.$refs.splInner.panes[0].size = ret.topPane;
      }
      if (this.showBottomPane()) {
        let bottomPaneSize;
        if (!this.chatStore.isEditingBigText) {
          bottomPaneSize = ret.bottomPane;
        } else {
          bottomPaneSize = ret.bottomPaneBig;
        }
        this.$refs.splInner.panes[this.$refs.splInner.panes.length - 1].size = bottomPaneSize;
      }
      this.setMiddlePane(ret);
    },
    setMiddlePane(ret) {
      let middleSize = 100; // percents
      let middlePaneIndex = 0;
      if (this.showTopPane()) {
        middleSize -= ret.topPane;
        middlePaneIndex = 1;
      }
      if (this.showBottomPane()) {
        let bottomPaneSize;
        if (!this.chatStore.isEditingBigText) {
          bottomPaneSize = ret.bottomPane;
        } else {
          bottomPaneSize = ret.bottomPaneBig;
        }
        middleSize -= bottomPaneSize;
      }
      this.$refs.splInner.panes[middlePaneIndex].size = middleSize;
    },

    onPanelAdd() {
      // console.debug("On panel add", this.$refs.splOuter.panes);
      this.$nextTick(() => {
        const stored = this.getStored();
        // console.debug("Restoring on add", stored)
        this.restorePanelsSize(stored);
      })

    },
    onPanelRemove() {
      // console.debug("On panel removed", this.$refs.splOuter.panes);
      this.$nextTick(() => {
        const stored = this.getStored();
        // console.debug("Restoring on remove", stored)
        this.restorePanelsSize(stored);
      })
    },
    onPanelResized() {
      this.$nextTick(() => {
        this.saveToStored(this.prepareForStore());
      })
    },

  },
  watch: {
    '$route': {
      handler: function (newValue, oldValue) {
        if (isChatRoute(newValue)) {
          if (newValue.params.id != oldValue.params.id) {
            console.debug("Chat id has been changed", oldValue.params.id, "->", newValue.params.id);
            if (hasLength(newValue.params.id)) {
              this.isSwitching = true; // used to prevent opening ChatVideo with old (previous) chatDto that contains old chatId
              this.partialReset();
              this.onProfileSet().finally(()=>{
                this.isSwitching = false;
              });
            }
          }
        }
      }
    },
    'chatStore.isEditingBigText': {
      handler: function (newValue, oldValue) {
        const stored = this.getStored();
        this.setMiddlePane(stored);
      }
    }
  },
  created() {

  },
  async mounted() {
    this.chatStore.title = `Chat #${this.chatId}`;
    this.chatStore.chatUsersCount = 0;
    this.chatStore.isShowSearch = true;
    this.chatStore.showChatEditButton = false;

    if (this.chatStore.currentUser) {
      await this.onProfileSet();
    }
    this.chatStore.showCallManagement = true;

    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLogout);
    bus.on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
    bus.on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    bus.on(FOCUS, this.onFocus);
    bus.on(USER_TYPING, this.onUserTyping);
    bus.on(MESSAGE_BROADCAST, this.onUserBroadcast);
    bus.on(CHAT_EDITED, this.onChatChange);
    bus.on(CHAT_DELETED, this.onChatDelete);
    bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);

    writingUsersTimerId = setInterval(()=>{
      const curr = + new Date();
      this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
      if (this.writingUsers.length == 0) {
        this.chatStore.moreImportantSubtitleInfo = null;
      }
    }, 500);

  },
  beforeUnmount() {
    this.graphQlUnsubscribe();

    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLogout);
    bus.off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
    bus.off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);
    bus.off(FOCUS, this.onFocus);
    bus.off(USER_TYPING, this.onUserTyping);
    bus.off(MESSAGE_BROADCAST, this.onUserBroadcast);
    bus.off(CHAT_EDITED, this.onChatChange);
    bus.off(CHAT_DELETED, this.onChatDelete);
    bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);

    this.chatStore.title = null;
    setTitle(null);
    this.chatStore.avatar = null;
    this.chatStore.showGoToBlogButton = null;
    this.chatStore.showCallManagement = false;

    this.chatStore.isShowSearch = false;
    this.chatStore.chatUsersCount = 0;

    this.partialReset();
    clearInterval(writingUsersTimerId);

    this.isSwitching = false;
    this.chatStore.isEditingBigText = false;
  }
}
</script>


<style scoped lang="stylus">
@import "pinned.styl"

.pinned-promoted {
  position: absolute
  z-index: 4;
}
.message-pane {
  position: relative // needed for the correct displaying .pinned-promoted
}
.message-pane-mobile {
    align-items: unset;
}

.new-fab {
  position: absolute
  bottom: 20px
  right: 20px
  z-index: 1000
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

    // apply only for Chrome
    @supports (not (-moz-appearance:button)) and (contain:paint) and (-webkit-appearance:none) {
      .v-alert__close {
          .v-btn__content {
              margin-top 3px
          }
      }
    }
  }

}
</style>
