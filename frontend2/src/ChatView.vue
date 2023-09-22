<template>
    <splitpanes class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar">
      <pane size="20">
        <ChatList/>
      </pane>
      <pane>
        <splitpanes class="default-theme" :dbl-click-splitter="false" horizontal>
            <pane style="width: 100%">
              <v-tooltip
                v-if="writingUsers.length || broadcastMessage"
                :model-value="showTooltip"
                activator=".message-edit-pane"
                location="bottom start"
              >
                <span v-if="!broadcastMessage">{{writingUsers.map(v=>v.login).join(', ')}} {{ $vuetify.locale.t('$vuetify.user_is_writing') }}</span>
                <span v-else v-html="broadcastMessage"></span>
              </v-tooltip>

              <div v-if="pinnedPromoted" :key="pinnedPromotedKey" class="pinned-promoted" :title="$vuetify.locale.t('$vuetify.pinned_message')">
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
            </pane>
          <pane size="25" class="message-edit-pane">
            <MessageEdit :chatId="this.chatId"/>
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
import {hasLength, offerToJoinToPublicChatStatus, setTitle} from "@/utils";
import bus, {
  FILE_CREATED, FILE_REMOVED, FILE_UPDATED, FOCUS, LOGGED_OUT,
  MESSAGE_ADD,
  MESSAGE_BROADCAST,
  MESSAGE_DELETED,
  MESSAGE_EDITED,
  PARTICIPANT_ADDED,
  PARTICIPANT_DELETED,
  PARTICIPANT_EDITED,
  PINNED_MESSAGE_PROMOTED,
  PINNED_MESSAGE_UNPROMOTED,
  PREVIEW_CREATED,
  PROFILE_SET,
  USER_TYPING,
  VIDEO_CALL_USER_COUNT_CHANGED
} from "@/bus/bus";
import {chat_list_name, chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";

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


export default {
  mixins: [
    heightMixin(),
    graphqlSubscriptionMixin('chatEvents'),
  ],
  data() {
    return {
      chatDto: chatDtoFactory(),
      pinnedPromoted: null,
      pinnedPromotedKey: +new Date(),
      writingUsers: [],
      showTooltip: true,
      broadcastMessage: null,
    }
  },
  components: {
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
      this.graphQlUnsubscribe();
    },
    fetchAndSetChat() {
      return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
        console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
        this.chatStore.title = data.name;
        setTitle(data.name);
        this.chatStore.avatar = data.avatar;
        this.chatStore.chatUsersCount = data.participantsCount;
        this.chatStore.showChatEditButton = data.canEdit;
        this.chatStore.canBroadcastTextMessage = data.canBroadcast;
        this.chatStore.tetATet = data.tetATet;
        if (data.blog) {
          this.chatStore.showGoToBlogButton = this.chatId;
        }
        this.chatDto = data;
      })
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
      return this.fetchAndSetChat().catch(reason => {
        if (reason.response.status == 404) {
          this.goToChatList();
          return Promise.reject();
        } else if (reason.response.status == offerToJoinToPublicChatStatus) {
          return axios.put(`/api/chat/${this.chatId}/join`).then(() => {
            return this.fetchAndSetChat();
          })
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
        this.fetchPromotedMessage();
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
  },
  watch: {
    async chatId(newVal, oldVal) {
      console.debug("Chat id has been changed", oldVal, "->", newVal);
      if (hasLength(newVal)) {
        await this.onProfileSet();
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

    writingUsersTimerId = setInterval(()=>{
      const curr = + new Date();
      this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
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

    this.chatStore.title = null;
    setTitle(null);
    this.chatStore.avatar = null;
    this.chatStore.showGoToBlogButton = null;
    this.chatStore.showCallButton = false;
    this.chatStore.showHangButton = false;
    this.chatStore.videoChatUsersCount = 0;
    this.chatStore.canMakeRecord = false;
    this.chatStore.canBroadcastTextMessage = false;
    this.chatStore.showRecordStartButton = false;
    this.chatStore.showRecordStopButton = false;
    this.chatStore.chatUsersCount = 0;
    this.chatStore.isShowSearch = true;
    this.chatStore.showChatEditButton = false;

    this.pinnedPromoted = null;
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
</style>

<style lang="stylus">
.pinned-promoted {
  .v-alert__content{
    text-overflow: ellipsis;
  }
}
</style>
