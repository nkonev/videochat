<template>
    <splitpanes class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar">
      <pane size="25" v-if="!isMobile()">
        <ChatList :embedded="true"/>
      </pane>
      <pane>
        <splitpanes class="default-theme" :dbl-click-splitter="false">
          <pane>
            <splitpanes class="default-theme" :dbl-click-splitter="false" horizontal @resize="onPanelResized($event)" @pane-add="onPanelAdd($event)" @pane-remove="onPanelRemove($event)">
              <pane v-if="shouldShowVideoOnTop()" min-size="15" :size="onTopVideoSize">
                <ChatVideo :chatDto="chatDto" :videoIsOnTop="videoIsOnTop()" />
              </pane>

              <pane style="width: 100%" :class="isMobile() ? 'message-pane-mobile' : ''" :size="messageListSize">
                  <v-tooltip
                    v-if="broadcastMessage"
                    :model-value="showTooltip"
                    activator=".message-edit-pane"
                    location="bottom start"
                  >
                    <span v-html="broadcastMessage"></span>
                  </v-tooltip>

                  <div v-if="pinnedPromoted" :key="pinnedPromotedKey" :class="!isMobile() ? 'pinned-promoted' : ['pinned-promoted', 'pinned-promoted-mobile']" :title="$vuetify.locale.t('$vuetify.pinned_message')">
                    <v-alert
                      closable
                      color="red-lighten-4"
                      elevation="2"
                      density="compact"
                    >
                      <router-link :to="getPinnedRouteObject(pinnedPromoted)" class="pinned-text" v-html="pinnedPromoted.text">
                      </router-link>
                    </v-alert>
                  </div>

                  <MessageList :chatDto="chatDto"/>

                  <v-btn v-if="isMobile()" variant="elevated" color="primary" icon="mdi-plus" class="new-fab" @click="openNewMessageDialog()"></v-btn>
              </pane>
              <pane class="message-edit-pane" v-if="!isMobile()" :size="messageEditSize">
                <MessageEdit :chatId="this.chatId"/>
              </pane>
            </splitpanes>
          </pane>
          <pane v-if="videoIsAtSide() && isAllowedVideo()" min-size="15" size="40">
            <ChatVideo :chatDto="chatDto" :videoIsOnTop="videoIsOnTop()"/>
          </pane>

        </splitpanes>
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
import {hasLength, isChatRoute, setTitle} from "@/utils";
import bus, {
  CHAT_DELETED,
  CHAT_EDITED,
  FILE_CREATED, FILE_REMOVED, FILE_UPDATED, FOCUS, LOGGED_OUT,
  MESSAGE_ADD,
  MESSAGE_BROADCAST,
  MESSAGE_DELETED,
  MESSAGE_EDITED, MESSAGE_EDITING_BIG_TEXT_START, MESSAGE_EDITING_END, OPEN_EDIT_MESSAGE,
  PARTICIPANT_ADDED,
  PARTICIPANT_DELETED,
  PARTICIPANT_EDITED,
  PINNED_MESSAGE_PROMOTED,
  PINNED_MESSAGE_UNPROMOTED,
  PREVIEW_CREATED,
  PROFILE_SET, REFRESH_ON_WEBSOCKET_RESTORED,
  USER_TYPING,
  VIDEO_CALL_USER_COUNT_CHANGED, VIDEO_DIAL_STATUS_CHANGED
} from "@/bus/bus";
import {chat_list_name, chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import ChatVideo from "@/ChatVideo.vue";
import videoPositionMixin from "@/mixins/videoPositionMixin";

const chatDtoFactory = () => {
  return {
    participantIds:[],
    participants:[],
  }
}

const getChatEventsData = (message) => {
  return message.data?.chatEvents
};

let writingUsersTimerId;

const MESSAGE_EDIT_PANE_SIZE_INITIAL = 20; // percents
const MESSAGE_EDIT_PANE_SIZE_EXPANDED = 60; // percents
const ONTOP_VIDEO_PANE_SIZE = 30;

export default {
  mixins: [
    heightMixin(),
    graphqlSubscriptionMixin('chatEvents'),
    videoPositionMixin(),
  ],
  data() {
    return {
      chatDto: chatDtoFactory(),
      pinnedPromoted: null,
      pinnedPromotedKey: +new Date(),
      writingUsers: [],
      showTooltip: true,
      broadcastMessage: null,
      messageEditSize: MESSAGE_EDIT_PANE_SIZE_INITIAL, // percents
      prevMessageEditSize: null, // percents
      onTopVideoSize: ONTOP_VIDEO_PANE_SIZE,
      messageListSize: null,
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
      this.getInfo();
      this.graphQlSubscribe();
    },
    onLogout() {
      this.partialReset();
      this.graphQlUnsubscribe();
    },
    fetchAndSetChat() {
      return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
        console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
        this.commonChatEdit(data);
        this.chatStore.tetATet = data.tetATet;
        this.chatDto = data;
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
      this.fetchPromotedMessage();
      return this.fetchAndSetChat().catch(reason => {
        if (reason.response.status == 404) {
          this.goToChatList();
          return Promise.reject();
        } else {
          return Promise.resolve();
        }
      }).then(() => {
        // async call
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
                                      }
                                      count
                                      fileItemUuid
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
      }
    },
    isVideoRoute() {
      return this.$route.name == videochat_name
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
      return this.chatStore.currentUser && this.$route.name == videochat_name && this.chatDto?.participantIds?.length
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
      this.chatStore.videoChatUsersCount=0;
      this.chatStore.canMakeRecord=false;
      this.pinnedPromoted=null;
      this.chatStore.canBroadcastTextMessage = false;
      this.chatStore.showRecordStartButton = false;
      this.chatStore.showRecordStopButton = false;
      this.chatStore.showChatEditButton = false;
    },
    onChatDialStatusChange(dto) {
      if (this.chatDto.tetATet) {
        for (const videoDialChanged of dto.dials) {
          if (this.chatStore.currentUser.id != videoDialChanged.userId) {
            this.chatStore.shouldPhoneBlink = videoDialChanged.status;
          }
        }
      }
    },
    openNewMessageDialog() { // on mobile OPEN_EDIT_MESSAGE with the null argument
      bus.emit(OPEN_EDIT_MESSAGE, null);
    },
    onEditingBigTextStart() {
      if (!this.prevMessageEditSize && !this.isMobile()) {
        this.prevMessageEditSize = this.messageEditSize;
        this.messageEditSize = MESSAGE_EDIT_PANE_SIZE_EXPANDED;
        this.$nextTick(()=>{
          this.messageListSize = this.shouldShowVideoOnTop() ? (100 - this.messageEditSize - this.onTopVideoSize) : (100 - this.messageEditSize);
        })
      }
    },
    onEditingEnd() {
      if (this.prevMessageEditSize && !this.isMobile()) {
        this.messageEditSize = this.prevMessageEditSize;
        this.prevMessageEditSize = null;
        this.$nextTick(()=>{
          this.messageListSize = this.shouldShowVideoOnTop() ? (100 - this.messageEditSize - this.onTopVideoSize) : (100 - this.messageEditSize);
        })
      }
    },
    onPanelResized(e) {
      if (!this.isMobile()) {
        //console.log(">>> onPanelResized", e)
        const pane = e[e.length - 1];
        this.messageEditSize = pane.size;
      }
    },
    shouldShowVideoOnTop() {
        return this.videoIsOnTop() && this.isAllowedVideo()
    },
    onPanelAdd(e) {
      if (!this.isMobile()) {
        const tmp = this.messageEditSize;
        this.messageEditSize = null;
        this.$nextTick(() => {
          this.messageEditSize = tmp;
          //console.log(">>> onPanelAdd", e, this.messageEditSize);
        }).then(() => {
          this.messageListSize = 100 - tmp - this.onTopVideoSize;
        })
      }
    },
    onPanelRemove(e) {
      if (!this.isMobile()) {
        const tmp = this.messageEditSize;
        this.messageEditSize = null;
        this.$nextTick(() => {
          this.messageEditSize = tmp;
          //console.log(">>> onPanelRemove", e, this.messageEditSize);
        }).then(() => {
          this.messageListSize = 100 - tmp;
        })
      }
    },
  },
  watch: {
    '$route': {
      handler: async function (newValue, oldValue) {
        if (isChatRoute(newValue)) {
          if (newValue.params.id != oldValue.params.id) {
            console.debug("Chat id has been changed", oldValue.params.id, "->", newValue.params.id);
            if (hasLength(newValue.params.id)) {
              await this.onProfileSet();
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

    if (this.chatStore.currentUser) {
      await this.onProfileSet();
    }

    this.chatStore.showCallButton = true;
    this.chatStore.showHangButton = false;

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
    bus.on(MESSAGE_EDITING_BIG_TEXT_START, this.onEditingBigTextStart);
    bus.on(MESSAGE_EDITING_END, this.onEditingEnd);

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
    bus.off(MESSAGE_EDITING_BIG_TEXT_START, this.onEditingBigTextStart);
    bus.on(MESSAGE_EDITING_END, this.onEditingEnd);

    this.chatStore.title = null;
    setTitle(null);
    this.chatStore.avatar = null;
    this.chatStore.showGoToBlogButton = null;
    this.chatStore.showCallButton = false;
    this.chatStore.showHangButton = false;
    this.chatStore.isShowSearch = false;
    this.chatStore.chatUsersCount = 0;

    this.partialReset();
    clearInterval(writingUsersTimerId);
  }
}
</script>


<style scoped lang="stylus">
@import "pinned.styl"

.pinned-promoted {
  position: fixed
  z-index: 4;
  margin-right: 284px;
}
.pinned-promoted-mobile {
    margin-right: unset;
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
}
</style>
