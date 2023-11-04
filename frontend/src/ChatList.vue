<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="chat-list-items" class="my-chat-scroller" @scroll.passive="onScroll">
            <div class="chat-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                class="list-item-prepend-spacer-16 pb-2"
                @contextmenu.stop="onShowContextMenu($event, item)"
                @click.prevent="openChat(item)"
                :href="getLink(item)"
            >
                <template v-slot:prepend v-if="hasLength(item.avatar)">
                    <v-badge
                        :color="getUserBadgeColor(item)"
                        dot
                        location="right bottom"
                        overlap
                        bordered
                        :model-value="item.online"
                    >
                      <span class="item-avatar">
                        <img :src="item.avatar">
                      </span>
                    </v-badge>
                </template>

                <template v-slot:default>
                    <v-list-item-title>
                        <span class="chat-name" :style="isSearchResult(item) ? {color: 'gray'} : {}" :class="getItemClass(item)" v-html="getChatName(item)"></span>
                        <v-badge v-if="item.unreadMessages" color="primary" inline :content="item.unreadMessages" class="mt-0" :title="$vuetify.locale.t('$vuetify.unread_messages')"></v-badge>
                        <v-badge v-if="item.videoChatUsersCount" color="success" icon="mdi-phone" inline  class="mt-0" :title="$vuetify.locale.t('$vuetify.call_in_process')"/>
                        <v-badge v-if="item.hasScreenShares" color="primary" icon="mdi-monitor-screenshot" inline  class="mt-0" :title="$vuetify.locale.t('$vuetify.screen_share_in_process')"/>
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
            <template v-if="items.length == 0 && !showProgress">
              <v-sheet class="mx-2">{{$vuetify.locale.t('$vuetify.chats_not_found')}}</v-sheet>
            </template>
            <div class="chat-last-element" style="min-height: 1px; background: white"></div>
      </v-list>
      <ChatListContextMenu
        ref="contextMenuRef"
        @editChat="this.editChat"
        @deleteChat="this.deleteChat"
        @leaveChat="this.leaveChat"
        @pinChat="this.pinChat"
        @removedFromPinned="this.removedFromPinned"
      />

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
  CHAT_ADD,
  CHAT_DELETED,
  CHAT_EDITED, CHAT_REDRAW,
  CLOSE_SIMPLE_MODAL,
  LOGGED_OUT,
  OPEN_CHAT_EDIT,
  OPEN_SIMPLE_MODAL,
  PROFILE_SET,
  REFRESH_ON_WEBSOCKET_RESTORED,
  SEARCH_STRING_CHANGED,
  UNREAD_MESSAGES_CHANGED,
  PARTICIPANT_CHANGED,
  VIDEO_CALL_SCREEN_SHARE_CHANGED,
  VIDEO_CALL_USER_COUNT_CHANGED
} from "@/bus/bus";
import {searchString, SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
  deepCopy,
  dynamicSortMultiple,
  findIndex,
  hasLength,
  isArrEqual, isChatRoute, publicallyAvailableForSearchChatsQuery, replaceInArray,
  replaceOrAppend,
  replaceOrPrepend,
  setTitle
} from "@/utils";
import Mark from "mark.js";
import ChatListContextMenu from "@/ChatListContextMenu.vue";
import userStatusMixin from "@/mixins/userStatusMixin";

const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'ChatList';

export default {
  mixins: [
    infiniteScrollMixin(scrollerName),
    heightMixin(),
    searchString(SEARCH_MODE_CHATS),
    userStatusMixin('tetATetInChatList'),
  ],
  props:['embedded'],
  data() {
    return {
        pageTop: 0,
        pageBottom: 0,
        markInstance: null,
    }
  },
  computed: {
    ...mapStores(useChatStore),
      tetAtetParticipants() {
          return this.getTetATetParticipantIds(this.items);
      },
      showProgress() {
          return this.chatStore.progressCount > 0
      },
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
              .get(`/api/chat/get-page`, {params: {id: id, size: PAGE_SIZE, searchString: this.searchString}})
              .then(({data}) => data.page) - 1; // as in load() -> axios.get().then()
          if (this.pageTop == -1) {
              this.pageTop = 0
          }
          console.log("Set page top", this.pageTop, "for id", id);
      } else {
          const id = this.findBottomElementId();
          //console.log("Going to get bottom page", aDirection, id);
          this.pageBottom = await axios
              .get(`/api/chat/get-page`, {params: {id: id, size: PAGE_SIZE, searchString: this.searchString}})
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

          // replaceOrPrepend() and replaceOrAppend() for the situation when order has been changed on server,
          // e.g. some chat has been popped up on sever due to somebody updated it
          if (this.isTopDirection()) {
              replaceOrPrepend(this.items, items.reverse());
              this.sort(this.items); // sorts possibly wrong order after loading items, appeared on server while user was scrolling
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
          this.performMarking();
        }).finally(()=>{
          this.chatStore.decrementProgressCount();
          return this.$nextTick();
        })
    },
    afterScrollRestored(el) {
        el?.parentElement?.scrollBy({
          top: !this.isTopDirection() ? 10 : -10,
          behavior: "instant",
        });
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

    async onSearchStringChangedDebounced() {
      await this.onSearchStringChanged();
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
    onWsRestoredRefresh() {
      this.onSearchStringChanged();
    },
    async onProfileSet() {
      await this.reloadItems();
    },
    onLoggedOut() {
      this.graphQlUnsubscribe();
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
              'pinned-bold': item.pinned,
          }
    },
    getChatName(item) {
          let bldr = item.name;
          if (item.tetATet) {
              bldr += this.getUserName(item);
          }
          return bldr;
    },
    onShowContextMenu(e, menuableItem) {
        this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
    },
    openChat(item){
      this.chatStore.incrementProgressCount();
      const prev = deepCopy(this.$route.query);
      if (isChatRoute(this.$route)) {
        delete prev[SEARCH_MODE_MESSAGES]
      }

      let promise;
      if (this.isSearchResult(item)) {
        promise = axios.put(`/api/chat/${item.id}/join`)
      } else {
        promise = Promise.resolve(true)
      }
      promise.then(() => {
        this.$router.push({ name: chat_name, params: {id: item.id}, query: prev }).finally(()=>{
          this.chatStore.decrementProgressCount();
        })
      })
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
    setTopTitle() {
        this.chatStore.title = this.$vuetify.locale.t('$vuetify.chats');
        setTitle(this.$vuetify.locale.t('$vuetify.chats'));
    },
    getTetATetParticipantIds(items) {
      if (!items) {
          return [];
      }
      const tmps = deepCopy(items);
      return tmps.filter((item) => item.tetATet).map((item) => item.participantIds.filter((pId) => pId != this.chatStore.currentUser?.id)[0]);
    },
    getUserIdsSubscribeTo() {
      return this.tetAtetParticipants
    },
    onUserStatusChanged(rawData) {
          const dtos = rawData?.data?.userEvents;
          if (dtos) {
              this.items.forEach(item => {
                if (item.tetATet) {
                  dtos.forEach(dtoItem => {
                    if (dtoItem.online !== null && item.participants.filter((p) => p.id == dtoItem.userId).length) {
                      item.online = dtoItem.online;
                    }
                    if (dtoItem.isInVideo !== null && item.participants.filter((p)=> p.id == dtoItem.userId).length) {
                      item.isInVideo = dtoItem.isInVideo;
                    }
                  })
                }
              })
          }
    },
    onChangeUnreadMessages(dto) {
          const chatId = dto.chatId;
          let idxOf = findIndex(this.items, {id: chatId});
          if (idxOf != -1) {
              this.items[idxOf].unreadMessages = dto.unreadMessages;
              this.items[idxOf].lastUpdateDateTime = dto.lastUpdateDateTime;

              this.sort(this.items);
          } else {
              console.log("Not found to update unread messages", dto)
          }
    },
    sort(items) {
          // also see in chat/db/chat.go:GetChatsByLimitOffset
          items.sort(dynamicSortMultiple("-pinned", "-lastUpdateDateTime", "-id"))
    },
    performMarking() {
        this.$nextTick(()=>{
            if (hasLength(this.searchString)) {
                this.markInstance.unmark();
                this.markInstance.mark(this.searchString);
            }
        })
    },
    onScrollCallback() {
          const isScrolledToTop = this.isScrolledToTop();
          if (!isScrolledToTop) {
              // during scrolling we disable adding new elements, so some messages can appear on server, so
              // we set loadedTop to false in order to force infiniteScrollMixin to fetch new messages during scrollTop()
              this.loadedTop = false;
              // see also this.sort(this.items) in load()
          }
    },
    isScrolledToTop() {
          if (this.scrollerDiv) {
              return Math.abs(this.scrollerDiv.scrollTop) < SCROLLING_THRESHHOLD
          } else {
              return false
          }
    },
    addItem(dto) {
        const isScrolledToTop = this.isScrolledToTop();
        const emptySearchString = !hasLength(this.searchString);
        if (isScrolledToTop && emptySearchString) {
            console.log("Adding item", dto);
            this.transformItem(dto);
            this.items.unshift(dto);
            this.sort(this.items);
            this.performMarking();
        } else {
            console.log("Skipping", dto, isScrolledToTop, emptySearchString)
        }
    },
    changeItem(dto) {
          console.log("Replacing item", dto);
          this.transformItem(dto);
          if (this.hasItem(dto)) {
              replaceInArray(this.items, dto);
          } else {
              this.items.unshift(dto);
          }
          this.sort(this.items);
          this.performMarking();
    },
    redrawItem(dto) {
      if (this.searchString == publicallyAvailableForSearchChatsQuery) {
          this.changeItem(dto)
      }
    },
    removeItem(dto) {
          if (this.hasItem(dto)) {
              console.log("Removing item", dto);
              const idxToRemove = findIndex(this.items, dto);
              this.items.splice(idxToRemove, 1);
          } else {
              console.log("Item was not be removed", dto);
          }
    },
      // does should change items list (new item added to visible part or not for example)
    hasItem(item) {
          let idxOf = findIndex(this.items, item);
          return idxOf !== -1;
    },
    onUserProfileChanged(user) {
      this.items.forEach(item => {
        replaceInArray(item.participants, user); // replaces participants of "normal" chat
        if (item.tetATet && this.getTetATetParticipantIds([item]).includes(user.id)) { // replaces content of tet-a-tet. It's better to move it to chat
          item.avatar = user.avatar;
          item.name = user.login;
          item.shortInfo = user.shortInfo;
        }
      });
    },
    onVideoCallChanged(dto) {
          this.items.forEach(item => {
              if (item.id == dto.chatId) {
                  item.videoChatUsersCount = dto.usersCount;
              }
          });
    },
    onVideoScreenShareChanged(dto) {
          this.items.forEach(item => {
              if (item.id == dto.chatId) {
                  item.hasScreenShares = dto.hasScreenShares;
              }
          });
    },

  },
  components: {
    ChatListContextMenu
  },
  created() {
    this.onSearchStringChangedDebounced = debounce(this.onSearchStringChangedDebounced, 700, {leading:false, trailing:true})
  },
  watch: {
      '$vuetify.locale.current': {
          handler: function (newValue, oldValue) {
              this.setTopTitle();
          },
      },
      tetAtetParticipants: function(newValue, oldValue) {
          if (newValue.length == 0) {
              this.graphQlUnsubscribe();
          } else {
              if (!isArrEqual(oldValue, newValue)) {
                  this.graphQlSubscribe();
              }
          }
      },
  },
  async mounted() {
    this.markInstance = new Mark("div#chat-list-items .chat-name");
    this.setTopTitle();

    if (this.canDrawChats()) {
      await this.onProfileSet();
    }

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChangedDebounced);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);
    bus.on(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
    bus.on(CHAT_ADD, this.addItem);
    bus.on(CHAT_EDITED, this.changeItem);
    bus.on(CHAT_REDRAW, this.redrawItem);
    bus.on(CHAT_DELETED, this.removeItem);
    bus.on(PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.on(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);

    if (this.$route.name == chat_list_name) {
      this.chatStore.isShowSearch = true;
      this.chatStore.searchType = SEARCH_MODE_CHATS;
    }
  },

  beforeUnmount() {
    this.graphQlUnsubscribe();
    this.uninstallScroller();

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChangedDebounced);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
    bus.off(CHAT_ADD, this.addItem);
    bus.off(CHAT_EDITED, this.changeItem);
    bus.off(CHAT_REDRAW, this.redrawItem);
    bus.off(CHAT_DELETED, this.removeItem);
    bus.off(PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.off(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);

    setTitle(null);
    this.chatStore.title = null;

    this.chatStore.isShowSearch = false;
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
@import "itemAvatar.styl"

.pinned-bold {
    font-weight bold
}
</style>
