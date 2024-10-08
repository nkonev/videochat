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
                :active="isActiveChat(item)"
                color="primary"
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
                        <span class="chat-name" :style="getStyle(item)" :class="getItemClass(item)" v-html="getChatName(item)"></span>
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
        @markAsRead="markAsRead"
        @markAsReadAll="markAsReadAll"
      />

  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {
    directionBottom,
} from "@/mixins/infiniteScrollMixin";
import {chat, chat_list_name, chat_name, userIdHashPrefix, videochat_name} from "@/router/routes";
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
    CO_CHATTED_PARTICIPANT_CHANGED,
    VIDEO_CALL_SCREEN_SHARE_CHANGED,
    VIDEO_CALL_USER_COUNT_CHANGED, FOCUS
} from "@/bus/bus";
import {searchString, SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
    deepCopy,
    dynamicSortMultiple,
    findIndex,
    hasLength,
    isSetEqual, isChatRoute, publicallyAvailableForSearchChatsQuery, replaceInArray,
    replaceOrAppend,
    replaceOrPrepend,
    setTitle, getLoginColoredStyle
} from "@/utils";
import Mark from "mark.js";
import ChatListContextMenu from "@/ChatListContextMenu.vue";
import userStatusMixin from "@/mixins/userStatusMixin";
import MessageItemContextMenu from "@/MessageItemContextMenu.vue";

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
        paginationToken: "",
        markInstance: null,
        routeName: null,
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
    recreatePaginationToken() {
      axios.get("/api/chat/recreate-pagination-token", {
          params: {
              paginationToken: this.paginationToken,
              searchString: this.searchString,
              direction: this.aDirection,
              topElementId: this.findTopElementId(),
              bottomElementId: this.findBottomElementId(),
          },
      }).then((res) => {
          this.paginationToken = res.data.paginationToken;
      })
    },
    reduceBottom() {
        console.log("reduceBottom");
        this.items = this.items.slice(0, this.getReduceToLength());
        this.recreatePaginationToken();
    },
    reduceTop() {
        console.log("reduceTop");
        this.items = this.items.slice(-this.getReduceToLength());
        this.recreatePaginationToken();
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
    async load() {
      if (!this.canDrawChats()) {
        return Promise.resolve()
      }

      this.chatStore.incrementProgressCount();
      return axios.get(`/api/chat`, {
        params: {
          paginationToken: this.paginationToken,
          searchString: this.searchString,
          direction: this.aDirection,
          topElementId: this.findTopElementId(),
          bottomElementId: this.findBottomElementId(),
        },
      })
        .then((res) => {
          this.paginationToken = res.data.paginationToken;
          const items = res.data.data;
          console.log("Get items in ", scrollerName, items, "direction", this.aDirection);
          items.forEach((item) => {
                this.transformItem(item);
          });

          // replaceOrPrepend() and replaceOrAppend() for the situation when order has been changed on server,
          // e.g. some chat has been popped up on sever due to somebody updated it
          if (this.isTopDirection()) {
              replaceOrPrepend(this.items, items);
              // sorts possibly wrong order after loading items, appeared on server while user was scrolling
              // it makes sense only when user scrolls to top - in order to have more or less "fresh" view
              this.sort(this.items);
          } else {
              replaceOrAppend(this.items, items);
          }

          if (items.length < res.data.pageSize) {
            if (this.isTopDirection()) {
              this.loadedTop = true;
            } else {
              this.loadedBottom = true;
            }
          }
          this.performMarking();
          this.requestInVideo();
        }).finally(()=>{
          this.chatStore.decrementProgressCount();
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

      this.paginationToken = "";
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
      this.graphQlUserStatusUnsubscribe();
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

      this.$router.push({ name: chat_name, params: {id: item.id}, query: prev }).finally(()=>{
        this.chatStore.decrementProgressCount();
      })
    },
    getLink(item) {
          return chat + "/" + item.id
    },
    isActiveChat(item) {
        if (!this.isMobile() && this.$route.name == chat_name || this.$route.name == videochat_name) {
            return this.$route.params.id == item.id
        } else {
            return false
        }
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
          // console.log("Will add participants to chat", chatId);
          bus.emit(OPEN_CHAT_EDIT, chat);
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
      const tetATets = tmps.filter((item) => item.tetATet).flatMap((item) => item.participantIds);
      const uniq = [...new Set(tetATets)];
      const sorted = uniq.sort();
      return sorted
    },
    getUserIdsSubscribeTo() {
      return this.tetAtetParticipants
    },
    isNormalTetAtTet(item) {
        return item.participantIds.length == 2
    },
    withMyselfTetATet(item) {
        return item.participantIds.length == 1
    },
    filterOutMe(item) {
        return item.participants.filter((p) => p.id != this.chatStore.currentUser?.id)
    },
    onUserStatusChanged(dtos) {
          if (dtos) {
              this.items.forEach(item => {
                if (item.tetATet) {
                  dtos.forEach(dtoItem => {
                      if (this.isNormalTetAtTet(item)) { // normal tet-a-tet
                          if (dtoItem.online !== null && this.filterOutMe(item).filter((p) => p.id == dtoItem.userId).length) {
                              item.online = dtoItem.online;
                          }
                          if (dtoItem.isInVideo !== null && this.filterOutMe(item).filter((p)=> p.id == dtoItem.userId).length) {
                              item.isInVideo = dtoItem.isInVideo;
                          }
                      } else if (this.withMyselfTetATet(item)) { // tet-a-tet chat with user himself
                          if (dtoItem.online !== null && item.participants.filter((p) => p.id == dtoItem.userId).length) {
                              item.online = dtoItem.online;
                          }
                          if (dtoItem.isInVideo !== null && item.participants.filter((p)=> p.id == dtoItem.userId).length) {
                              item.isInVideo = dtoItem.isInVideo;
                          }
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
        } else if (isScrolledToTop) { // like in UserList.vue
            axios.post(`/api/chat/filter`, {
                searchString: this.searchString,
                chatId: dto.id
            }).then(({data}) => {
                if (data.found) {
                    console.log("Adding item", dto);
                    this.transformItem(dto);
                    this.items.unshift(dto);
                    this.sort(this.items);
                    this.performMarking();
                }
            })
        } else {
            console.log("Skipping", dto, isScrolledToTop, emptySearchString)
        }
    },
    changeItem(dto) {
          console.log("Replacing item", dto);
          this.transformItem(dto);

          let idxOf = findIndex(this.items, dto);
          if (idxOf !== -1) { // hasItem()
              const changedDto = this.applyState(this.items[idxOf], dto);
              replaceInArray(this.items, changedDto);
          } else {
              this.items.unshift(dto); // used to/along with redraw a public chat when user leaves from it
          }
          this.sort(this.items);
          this.performMarking();
    },
    redrawItem(dto) {
      if (this.searchString == publicallyAvailableForSearchChatsQuery) {
          this.changeItem(dto)
      } else {
          this.removeItem(dto)
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
        if (item.tetATet) {
            if (this.isNormalTetAtTet(item)) { // replace for counterpart
                if (this.filterOutMe(item).map(p => p.id).includes(user.id)) { // replaces content of tet-a-tet. It's better to move it to chat
                    item.avatar = user.avatar;
                    item.name = user.login;
                    item.shortInfo = user.shortInfo;
                    item.loginColor = user.loginColor;
                }
            } else if (this.withMyselfTetATet(item)) { // replace for with himself tet-a-tet
                if (item.participants.map(p => p.id).includes(user.id)) {
                    item.avatar = user.avatar;
                    item.name = user.login;
                    item.shortInfo = user.shortInfo;
                    item.loginColor = user.loginColor;
                }
            }
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
    getStyle(item) {
        let obj = {};
        if (item.tetATet) {
            obj = getLoginColoredStyle(item)
        }
        if (this.isSearchResult(item)) {
            obj.color = 'gray';
        }
        return obj;
    },

    onFocus() {
      if (this.chatStore.currentUser && this.items) {
          const list = this.items.filter(item => item.tetATet).flatMap(item => item.participantIds);
          const uniqueUserIds = [...new Set(list)];
          const joined = uniqueUserIds.join(",");
          axios.put(`/api/aaa/user/request-for-online`, null, {
              params: {
                  userId: joined
              },
          }).then(()=>{
              this.requestInVideo();
          })
      }
    },
    requestInVideo() {
        this.$nextTick(()=>{
            const userIds = this.tetAtetParticipants;
            const joined = userIds.join(",");

            axios.put("/api/video/user/request-in-video-status", null, {
                params: {
                    userId: joined
                },
            });
        })
    },
    hasItems() {
        return !!this.items?.length
    },
    markAsRead(item) {
      axios.put(`/api/chat/${item.id}/read`)
    },
    markAsReadAll(item) {
      axios.put(`/api/chat/read`)
    },
  },
  components: {
    MessageItemContextMenu,
    ChatListContextMenu
  },
  created() {
    this.routeName = this.$route.name;
    this.onSearchStringChangedDebounced = debounce(this.onSearchStringChangedDebounced, 700, {leading:false, trailing:true})
  },
  watch: {
      '$vuetify.locale.current': {
          handler: function (newValue, oldValue) {
            if (this.routeName == chat_list_name) {
              this.setTopTitle();
            }
          },
      },
      tetAtetParticipants: function(newValue, oldValue) {
          if (oldValue.length !== 0 && newValue.length === 0) {
              this.graphQlUserStatusUnsubscribe();
          } else {
              if (!isSetEqual(oldValue, newValue)) {
                  this.graphQlUserStatusSubscribe();
              }
          }
      },
  },
  async mounted() {
    this.markInstance = new Mark("div#chat-list-items .chat-name");

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
    bus.on(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.on(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);
    bus.on(FOCUS, this.onFocus);

    if (this.routeName == chat_list_name) {
      this.setTopTitle();
      this.chatStore.isShowSearch = true;
      this.chatStore.searchType = SEARCH_MODE_CHATS;
    }
  },

  beforeUnmount() {
    this.graphQlUserStatusUnsubscribe();
    this.uninstallScroller();

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChangedDebounced);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
    bus.off(CHAT_ADD, this.addItem);
    bus.off(CHAT_EDITED, this.changeItem);
    bus.off(CHAT_REDRAW, this.redrawItem);
    bus.off(CHAT_DELETED, this.removeItem);
    bus.off(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.off(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);
    bus.off(FOCUS, this.onFocus);

    if (this.routeName == chat_list_name) {
      setTitle(null);
      this.chatStore.title = null;

      this.chatStore.isShowSearch = false;
    }
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
