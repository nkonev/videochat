<template>
    <splitpanes class="default-theme" :dbl-click-splitter="false" :style="heightWithoutAppBar">
      <pane size="20">
        <ChatList/>
      </pane>
      <pane>
        <splitpanes class="default-theme" :dbl-click-splitter="false" horizontal>
            <pane>
                <MessageList :chatDto="chatDto"/>
            </pane>
          <pane size="25">
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
import {hasLength, offerToJoinToPublicChatStatus} from "@/utils";
import bus, {PROFILE_SET, VIDEO_CALL_USER_COUNT_CHANGED} from "@/bus/bus";
import {chat_list_name} from "@/router/routes";

const webSplitpanesCss = () => import('splitpanes/dist/splitpanes.css');
const mobileSplitpanesCss = () => import("@/splitpanes-mobile.scss");

const chatDtoFactory = () => {
  return {
    participantIds:[],
    participants:[],
  }
}

export default {
  mixins: [
    heightMixin(),
  ],
  data() {
    return {
      chatDto: chatDtoFactory(),
      pinnedPromoted: null,
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
      this.getInfo()
    },
    fetchAndSetChat() {
      return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
        console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
        this.chatStore.title = data.name;
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
    if (this.isMobile()) {
      mobileSplitpanesCss()
    } else {
      webSplitpanesCss()
    }
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
  },
  beforeUnmount() {
    bus.off(PROFILE_SET, this.onProfileSet);
    this.chatStore.title = null;
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
  }
}
</script>


<style>

</style>
