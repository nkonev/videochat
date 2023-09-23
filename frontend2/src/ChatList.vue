<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="chat-list-items" class="my-chat-scroller" @scroll.passive="onScroll">
            <div class="chat-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                class="list-item-prepend-spacer-16"
                @contextmenu.prevent="onShowContextMenu($event, item)"
                @click.prevent="openChat(item)"
                :href="getLink(item)"
            >
                <template v-slot:prepend v-if="hasLength(item.avatar)">
                    <v-badge
                        v-if="item.avatar"
                        color="success accent-4"
                        dot
                        location="right bottom"
                        overlap
                        bordered
                        :model-value="item.online"
                    >
                        <v-avatar :image="item.avatar"></v-avatar>
                    </v-badge>
                </template>

                <template v-slot:default>
                    <v-list-item-title>
                        <span class="chat-name min-height" :style="isSearchResult(item) ? {color: 'gray'} : {}" :class="getItemClass(item)" v-html="getChatName(item)"></span>
                        <v-badge v-if="item.unreadMessages" inline :content="item.unreadMessages" class="mt-0" :title="$vuetify.locale.t('$vuetify.unread_messages')"></v-badge>
                        <v-badge v-if="item.videoChatUsersCount" color="success" icon="mdi-phone" inline  class="mt-0" :title="$vuetify.locale.t('$vuetify.call_in_process')"/>
                        <v-badge v-if="item.hasScreenShares" icon="mdi-monitor-screenshot" inline  class="mt-0" :title="$vuetify.locale.t('$vuetify.screen_share_in_process')"/>
                        <v-badge v-if="item.blog" color="grey" icon="mdi-postage-stamp" inline  class="mt-0" :title="$vuetify.locale.t('$vuetify.blog')"/>
                    </v-list-item-title>
                    <v-list-item-subtitle :style="isSearchResult(item) ? {color: 'gray'} : {}">
                        {{ printParticipants(item) }}
                    </v-list-item-subtitle>
                </template>

                <template v-slot:append v-if="!isMobile() && !embedded">
                    <v-list-item-action>
                            <template v-if="!item.isResultFromSearch">
                                <v-btn variant="flat" icon v-if="item.pinned" @click.stop.prevent="removedFromPinned(item)" :title="$vuetify.locale.t('$vuetify.remove_from_pinned')"><v-icon size="large">mdi-pin-off-outline</v-icon></v-btn>
                                <v-btn variant="flat" icon v-else @click.stop.prevent="pinChat(item)" :title="$vuetify.locale.t('$vuetify.pin_chat')"><v-icon size="large">mdi-pin</v-icon></v-btn>
                            </template>
                            <v-btn variant="flat" icon v-if="item.canEdit" @click.stop.prevent="editChat(item)" :title="$vuetify.locale.t('$vuetify.edit_chat')"><v-icon color="primary" size="large">mdi-lead-pencil</v-icon></v-btn>
                            <v-btn variant="flat" icon v-if="item.canDelete" @click.stop.prevent="deleteChat(item)" :title="$vuetify.locale.t('$vuetify.delete_chat')"><v-icon color="red" size="large">mdi-delete</v-icon></v-btn>
                            <v-btn variant="flat" icon v-if="item.canLeave" @click.stop.prevent="leaveChat(item)" :title="$vuetify.locale.t('$vuetify.leave_chat')"><v-icon size="large">mdi-exit-run</v-icon></v-btn>
                    </v-list-item-action>
                </template>
            </v-list-item>
            <div class="chat-last-element" style="min-height: 1px; background: white"></div>
      </v-list>

  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {
    directionBottom,
    directionTop,
} from "@/mixins/infiniteScrollMixin";
import {chat, chat_list_name, chat_name} from "@/router/routes";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import heightMixin from "@/mixins/heightMixin";
import bus, {
    CLOSE_SIMPLE_MODAL,
    LOGGED_OUT,
    OPEN_CHAT_EDIT,
    OPEN_SIMPLE_MODAL,
    PROFILE_SET,
    SEARCH_STRING_CHANGED
} from "@/bus/bus";
import {searchString, goToPreserving, SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {hasLength, replaceOrAppend, replaceOrPrepend, setTitle} from "@/utils";

const PAGE_SIZE = 40;

const scrollerName = 'ChatList';

export default {
  mixins: [
    infiniteScrollMixin(scrollerName),
    heightMixin(),
    searchString(SEARCH_MODE_CHATS),
  ],
  props:['embedded'],
  data() {
    return {
        pageTop: 0,
        pageBottom: 0,
    }
  },
  computed: {
    ...mapStores(useChatStore),
  },

  methods: {
    hasLength,
    getMaxItemsLength() {
        return 240
    },
    getReduceToLength() {
        return 80 // in case numeric pages, should complement with getMaxItemsLength() and PAGE_SIZE
    },
    reduceBottom() {
        console.log("reduceBottom");
        this.items = this.items.slice(0, this.getReduceToLength());
        this.onReduce(directionBottom);
    },
    reduceTop() {
        console.log("reduceTop");
        this.items = this.items.slice(-this.getReduceToLength());
        this.onReduce(directionTop);
    },
    findBottomElementId() {
        return this.items[this.items.length-1]?.id
    },
    findTopElementId() {
        return this.items[0]?.id
    },
    saveScroll(top) {
        this.preservedScroll = top ? this.findTopElementId() : this.findBottomElementId();
        console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    async scrollTop() {
      return await this.$nextTick(() => {
          this.scrollerDiv.scrollTop = 0;
      });
    },
    initialDirection() {
      return directionBottom
    },
    async onFirstLoad() {
      this.loadedTop = true;
      await this.scrollTop();
    },
    async onReduce(aDirection) {
      if (aDirection == directionTop) { // became
          const id = this.findTopElementId();
          //console.log("Going to get top page", aDirection, id);
          this.pageTop = await axios
              .get(`/api/chat/get-page`, {params: {id: id, size: PAGE_SIZE,}})
              .then(({data}) => data.page) - 1; // as in load() -> axios.get().then()
          if (this.pageTop == -1) {
              this.pageTop = 0
          }
          console.log("Set page top", this.pageTop, "for id", id);
      } else {
          const id = this.findBottomElementId();
          //console.log("Going to get bottom page", aDirection, id);
          this.pageBottom = await axios
              .get(`/api/chat/get-page`, {params: {id: id, size: PAGE_SIZE,}})
              .then(({data}) => data.page);
          console.log("Set page bottom", this.pageBottom, "for id", id);
      }
    },
    async load() {
      if (!this.canDrawChats()) {
        return Promise.resolve()
      }

      this.chatStore.incrementProgressCount();
      const page = this.isTopDirection() ? this.pageTop : this.pageBottom;
      return axios.get(`/api/chat`, {
        params: {
          page: page,
          size: PAGE_SIZE,
          searchString: this.searchString,
        },
      })
        .then((res) => {
          const items = res.data.data;
          console.log("Get items in ", scrollerName, items, "page", page);
          items.forEach((item) => {
                this.transformItem(item);
          });

          // replaceOrPrepend() and replaceOrAppend() for the (future) situation when order has been changed on server,
          // e.g. some chat has been popped up on sever due to somebody updated it
          if (this.isTopDirection()) {
              replaceOrPrepend(this.items, items.reverse());
          } else {
              replaceOrAppend(this.items, items);
          }

          if (items.length < PAGE_SIZE) {
            if (this.isTopDirection()) {
              this.loadedTop = true;
            } else {
              this.loadedBottom = true;
            }
          } else {
            if (this.isTopDirection()) {
                this.pageTop -= 1;
                if (this.pageTop == -1) {
                    this.loadedTop = true;
                    this.pageTop = 0;
                }
            } else {
                this.pageBottom += 1;
            }
          }
        }).finally(()=>{
          this.chatStore.decrementProgressCount();
          return this.$nextTick();
        })
    },

    bottomElementSelector() {
      return ".chat-last-element"
    },
    topElementSelector() {
      return ".chat-first-element"
    },

    getItemId(id) {
      return 'chat-item-' + id
    },

    scrollerSelector() {
        return ".my-chat-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.pageTop = 0;
      this.pageBottom = 0;
    },

    async onSearchStringChanged() {
      // Fixes excess delayed (because of debounce) reloading of items when
      // 1. we've chosen __AVAILABLE_FOR_SEARCH
      // 2. then go to the Welcome
      // 3. without this change there will be excess delayed invocation
      // 4. but we've already destroyed this component, so it will be an error in the log
      if (this.isReady()) {
        await this.reloadItems();
      }
    },
    async onProfileSet() {
      await this.reloadItems();
    },
    onLoggedOut() {
      this.reset();
    },

    canDrawChats() {
      return !!this.chatStore.currentUser
    },
    isSearchResult(item) {
          return item?.isResultFromSearch === true
    },
    getItemClass(item) {
          return {
              'pinned': item.pinned,
          }
    },
    getChatName(item) {
          let bldr = item.name;
          if (!item.avatar && item.online) {
              bldr += (" (" + this.$vuetify.locale.t('$vuetify.user_online') + ")");
          }
          return bldr;
    },
    onShowContextMenu() {
          console.warn("Not implemented")
    },
    openChat(item){
          goToPreserving(this.$route, this.$router, { name: chat_name, params: { id: item.id}})
    },
    getLink(item) {
          return chat + "/" + item.id
    },
    printParticipants(chat) {
          if (hasLength(chat.shortInfo)) {
              return chat.shortInfo
          }
          let builder = "";
          if (chat.tetATet) {
              builder += this.$vuetify.locale.t('$vuetify.tet_a_tet');
          } else {
              const logins = chat.participants.map(p => p.login);
              builder += logins.join(", ")
          }
          if (this.isSearchResult(chat)) {
              builder = this.$vuetify.locale.t('$vuetify.this_is_search_result') + builder;
          }
          return builder;
    },
    pinChat(chat) {
          axios.put(`/api/chat/${chat.id}/pin`, null, {
              params: {
                  pin: true
              },
          });
    },
    removedFromPinned(chat) {
          axios.put(`/api/chat/${chat.id}/pin`, null, {
              params: {
                  pin: false
              },
          });
    },
    editChat(chat) {
          const chatId = chat.id;
          // console.log("Will add participants to chat", chatId);
          bus.emit(OPEN_CHAT_EDIT, chatId);
    },
    deleteChat(chat) {
          bus.emit(OPEN_SIMPLE_MODAL, {
              buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
              title: this.$vuetify.locale.t('$vuetify.delete_chat_title', chat.id),
              text: this.$vuetify.locale.t('$vuetify.delete_chat_text', chat.name),
              actionFunction: (that) => {
                  that.loading = true;
                  axios.delete(`/api/chat/${chat.id}`)
                      .then(() => {
                          bus.emit(CLOSE_SIMPLE_MODAL);
                      }).finally(()=>{
                      that.loading = false;
                  })
              }
          });
    },
    leaveChat(chat) {
          bus.emit(OPEN_SIMPLE_MODAL, {
              buttonName: this.$vuetify.locale.t('$vuetify.leave_btn'),
              title: this.$vuetify.locale.t('$vuetify.leave_chat_title', chat.id),
              text: this.$vuetify.locale.t('$vuetify.leave_chat_text', chat.name),
              actionFunction: (that) => {
                  that.loading = true;
                  axios.put(`/api/chat/${chat.id}/leave`)
                      .then(() => {
                          bus.emit(CLOSE_SIMPLE_MODAL);
                      }).finally(()=>{
                      that.loading = false;
                  })
              }
          });
    },
    transformItem(item) {
          item.online = false;
    },
    setTopTitle() {
        this.chatStore.title = this.$vuetify.locale.t('$vuetify.chats');
        setTitle(this.$vuetify.locale.t('$vuetify.chats'));
    },

  },
  created() {
    this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
  },
  watch: {
      '$vuetify.locale.current': {
          handler: function (newValue, oldValue) {
              this.setTopTitle();
          },
      },
  },
  async mounted() {
    this.setTopTitle();

    if (this.canDrawChats()) {
      await this.onProfileSet();
    }

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChanged);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);

    if (this.$route.name == chat_list_name) {
      this.chatStore.searchType = SEARCH_MODE_CHATS;
    }
  },

  beforeUnmount() {
    this.reset();
    this.uninstallScroller();
    console.log("Scroller", scrollerName, "has been uninstalled");

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChanged);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);

    setTitle(null);
    this.chatStore.title = null;
  }
}
</script>

<style lang="stylus">
.my-chat-scroller {
  height 100%
  overflow-y scroll !important
  display flex
  flex-direction column
}

</style>

<style lang="stylus" scoped>
.pinned {
    font-weight bold
}
</style>
