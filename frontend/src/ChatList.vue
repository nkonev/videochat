<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="chat-list-items" class="my-chat-scroller" @scroll.passive="onScroll">
            <div class="chat-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                :data-pinned="item.pinned"
                :data-last-update-date-time="item.lastUpdateDateTime"
                class="list-item-prepend-spacer pb-2 chat-item-root"
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
                    <v-list-item-subtitle :style="isSearchResult(item) ? {color: 'gray'} : {}" :class="getParticipantsClass(item)" v-html="printParticipants(item)">
                    </v-list-item-subtitle>
                    <v-list-item-subtitle v-if="shouldShowThirdLine(item)" class="subtitle-thin my-1" v-html="thirdLine(item)"></v-list-item-subtitle>
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
import {
  chat,
  chat_list_name,
  chat_name,
  chatIdHashPrefix, chatIdPrefix,
  videochat_name
} from "@/router/routes";
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
  VIDEO_CALL_USER_COUNT_CHANGED, USER_TYPING
} from "@/bus/bus";
import {searchString, SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
  deepCopy,
  dynamicSortMultiple,
  findIndex,
  hasLength,
  isSetEqual,
  isChatRoute,
  publicallyAvailableForSearchChatsQuery,
  replaceInArray,
  replaceOrAppend,
  replaceOrPrepend,
  setTitle,
  getLoginColoredStyle,
  isChatHash,
  upsertToWritingUsers,
  buildWritingUsersSubtitleInfo,
  filterOutOldWritingUsers,
} from "@/utils";
import Mark from "mark.js";
import ChatListContextMenu from "@/ChatListContextMenu.vue";
import userStatusMixin from "@/mixins/userStatusMixin";
import MessageItemContextMenu from "@/MessageItemContextMenu.vue";
import hashMixin from "@/mixins/hashMixin.js";
import {
  getTopChatPosition,
  removeTopChatPosition,
  setTopChatPosition
} from "@/store/localStore.js";
import onFocusMixin from "@/mixins/onFocusMixin.js";

const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'ChatList';

let writingUsersTimerId;

export default {
  mixins: [
    infiniteScrollMixin(scrollerName),
    hashMixin(),
    heightMixin(),
    searchString(SEARCH_MODE_CHATS),
    userStatusMixin('tetATetInChatList'),
    onFocusMixin(),
  ],
  props:['embedded'],
  data() {
    return {
        markInstance: null,
        routeName: null,
        startingFromItemIdTop: null,
        startingFromItemIdBottom: null
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
        this.items = this.items.slice(0, this.getReduceToLength()); // remove last from array, retain first N - reduce bottom on the page
        this.startingFromItemIdBottom = this.findBottomElementId();
    },
    reduceTop() {
        console.log("reduceTop");
        this.items = this.items.slice(-this.getReduceToLength());
        this.startingFromItemIdTop = this.findTopElementId();
    },
    enableHashInRoute() {
      return false
    },
    convertLoadedFromStoreHash(obj) {
      return chatIdHashPrefix + obj?.id
    },
    extractIdFromElementForStoring(element) {
      return {pinned: element.getAttribute("data-pinned") === "true", lastUpdateDateTime: element.getAttribute("data-last-update-date-time"), id: parseInt(this.getIdFromRouteHash(element.id))}
    },
    findBottomElementId() {
        if (!this.items.length) {
          return null
        }
        const e = this.items[this.items.length-1];
        return {pinned: e.pinned, lastUpdateDateTime: e.lastUpdateDateTime, id: e.id}
    },
    findTopElementId() {
        if (!this.items.length) {
          return null
        }
        const e = this.items[0];
        return {pinned: e.pinned, lastUpdateDateTime: e.lastUpdateDateTime, id: e.id}
    },
    updateTopAndBottomIds() {
      this.startingFromItemIdTop = this.findTopElementId();
      this.startingFromItemIdBottom = this.findBottomElementId();
    },
    saveScroll(top) {
        const obj = top ? this.findTopElementId() : this.findBottomElementId();
        if (obj) {
          this.preservedScroll = obj.id;
        } else {
          this.preservedScroll = null;
        }
        console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    async scrollTop() {
      removeTopChatPosition();
      return await this.$nextTick(() => {
          this.scrollerDiv.scrollTop = 0;
      });
    },
    initialDirection() {
      return directionBottom
    },
    async onFirstLoad(loadedResult) {
      await this.doScrollOnFirstLoad();
      if (loadedResult === true) {
        removeTopChatPosition();
      }
    },
    async fetchItems(startingFromItemId, reverse, includeStartingFrom) {
      const res = await axios.post(`/api/chat/search`, {
          startingFromItemId: startingFromItemId,
          includeStartingFrom: !!includeStartingFrom,
          size: PAGE_SIZE,
          reverse: reverse,
          searchString: this.searchString,
        }, {
          signal: this.requestAbortController.signal
      })
      const items = res.data.items;
      console.log("Get items in ", scrollerName, items, "direction", this.aDirection);
      items.forEach((item) => {
        this.transformItemOverride(item);
      });

      return items
    },
    async load() {
      if (!this.canDrawChats()) {
        return Promise.resolve()
      }

      const { startingFromItemId, hasHash } = this.prepareHashesForRequest();

      this.chatStore.incrementProgressCount();

      try {
        let items = await this.fetchItems(startingFromItemId, this.isTopDirection());
        if (hasHash) {
          const portion = await this.fetchItems(startingFromItemId, !this.isTopDirection(), true);
          items = portion.reverse().concat(items);
        }

        // replaceOrPrepend() and replaceOrAppend() for the situation when order has been changed on server,
        // e.g. some chat has been popped up on sever due to somebody updated it
        if (this.isTopDirection()) {
          replaceOrPrepend(this.items, items);
        } else {
          replaceOrAppend(this.items, items);
        }
        // sorts possibly wrong order after loading items, appeared on server while user was scrolling
        // it makes sense only when user scrolls to top - in order to have more or less "fresh" view
        this.sort(this.items);

        this.updateTopAndBottomIds();
        if (!this.isFirstLoad) {
          await this.clearRouteHash()
        }

        this.performMarking();

        this.requestStatuses();

        return Promise.resolve(true)
      } finally {
        this.chatStore.decrementProgressCount();
      }
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
      return chatIdPrefix + id
    },

    scrollerSelector() {
        return ".my-chat-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.startingFromItemIdTop = null;
      this.startingFromItemIdBottom = null;
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
      this.saveLastVisibleElement();
      this.doOnFocus();
    },
    async onProfileSet() {
      await this.initializeHashVariablesAndReloadItems();
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
        if (item.pinned) {
          return 'pinned-bold'
        } else{
          return 'chat-normal'
        }
    },
    getParticipantsClass(item) {
      let res = ['subtitle-thin'];
      if (item.lastMessagePreview) {
        res.push('my-1')
      }
      return res;
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
              signal: this.requestAbortController.signal
          });
    },
    removedFromPinned(chat) {
          axios.put(`/api/chat/${chat.id}/pin`, null, {
              params: {
                  pin: false
              },
              signal: this.requestAbortController.signal
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
                  // here is no signal: this.requestAbortController.signal
                  // because of it would cancel request on unmounting ChatList.vue because of unmounting CatView.vue
                  // due to navigating to ChatList.vue because the user isn't a chat member anymore
                  // the same is for leaveChat()
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
                  axios.put(`/api/chat/${chat.id}/leave`, null)
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
      console.log("Adding item", dto);
      this.transformItemOverride(dto);
      this.items.unshift(dto);
      this.sort(this.items);
      this.reduceListAfterAdd(false);
      this.updateTopAndBottomIds();
    },
    changeItem(dto) {
      console.log("Replacing item", dto);
      // this.transformItemOverride(dto); // not needed here because it resets online and in video. see applyStateOverride()
      replaceInArray(this.items, dto);
      this.sort(this.items);
      this.updateTopAndBottomIds();
    },
    removeItem(dto) {
      console.log("Removing item", dto);
      const idxToRemove = findIndex(this.items, dto);
      this.items.splice(idxToRemove, 1);
      this.updateTopAndBottomIds();
    },
    onNewChat(dto) {
        axios.post(`/api/chat/filter`, {
          searchString: this.searchString,
          pageSize: PAGE_SIZE,
          chatId: dto.id,
        }, {
          signal: this.requestAbortController.signal
        }).then(({data}) => {
          if (data.found) {
            this.addItem(dto);
            this.performMarking();
          } else {
            console.log("Skipping adding", dto)
          }
        })
    },
    onEditChat(dto) {
      // chat can change the position after chat_edited so we reflect it here
      let idxOf = findIndex(this.items, dto);
      if (idxOf !== -1) { // hasItem()
        const changedDto = this.applyStateOverride(this.items[idxOf], dto); // preserve online and isInVideo
        this.changeItem(changedDto);
      } else {
        this.addItem(dto); // used to/along with redraw a public chat when user leaves from it
      }
      this.performMarking();
    },
    transformItemOverride(item) {
      this.transformItem(item);
      item.writingUsers = [];
      item.usersWritingSubtitleInfo = null;
    },
    applyStateOverride(existing, newItem) {
      const s = this.applyState(existing, newItem);
      s.writingUsers = existing.writingUsers;
      s.usersWritingSubtitleInfo = existing.usersWritingSubtitleInfo;
      return s;
    },
    redrawItem(dto) {
      if (this.searchString == publicallyAvailableForSearchChatsQuery) {
          this.onEditChat(dto)
      } else {
          this.onDeleteChat(dto)
      }
    },
    onDeleteChat(dto) {
          if (this.hasItem(dto)) {
              this.removeItem(dto);
          } else {
              console.log("Item was not been removed", dto);
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
    requestStatuses() {
      this.$nextTick(()=> {
        const userIds = this.tetAtetParticipants;
        const joined = userIds.join(",");
        this.triggerUsesStatusesEvents(joined, this.requestAbortController.signal);
      })
    },
    onFocus() {
      if (this.chatStore.currentUser && this.items) {
        this.requestStatuses();

        if (this.isScrolledToTop()) {
          const topNElements = this.items.slice(0, PAGE_SIZE);
          axios.post(`/api/chat/fresh`, topNElements, {
            params: {
              size: PAGE_SIZE,
              searchString: this.searchString,
            },
            signal: this.requestAbortController.signal
          }).then((res)=>{
            if (!res.data.ok) {
              console.log("Need to update chats");
              this.reloadItems();
            } else {
              console.log("No need to update chats");
            }
          })
        }
      }
    },
    hasItems() {
        return !!this.items?.length
    },
    markAsRead(item) {
      axios.put(`/api/chat/${item.id}/read`, null, {
        signal: this.requestAbortController.signal
      })
    },
    markAsReadAll(item) {
      axios.put(`/api/chat/read`, null, {
        signal: this.requestAbortController.signal
      })
    },
    async doDefaultScroll() {
      await this.scrollTop(); // we need it to prevent browser's scrolling
    },
    getPositionFromStore() {
      return getTopChatPosition()
    },
    conditionToSaveLastVisible() {
      return !this.isScrolledToTop();
    },
    itemSelector() {
      return '.chat-item-root'
    },
    setPositionToStore(chatIdObj) {
      setTopChatPosition(chatIdObj)
    },
    beforeUnload() {
      this.saveLastVisibleElement();
    },
    isAppropriateHash(hash) {
      return isChatHash(hash)
    },

    onUserTyping(data) {
      const chatItem = this.items.find(it => it.id === data.chatId);
      if (chatItem) {
        upsertToWritingUsers(chatItem.writingUsers, data);
        chatItem.usersWritingSubtitleInfo = buildWritingUsersSubtitleInfo(chatItem.writingUsers, this.$vuetify);
      }
    },

    shouldShowThirdLine(item) {
      return !!item.lastMessagePreview || !!item.usersWritingSubtitleInfo
    },
    thirdLine(item) {
      if (item.usersWritingSubtitleInfo) {
        return item.usersWritingSubtitleInfo
      } else if (item.lastMessagePreview) {
        return item.lastMessagePreview
      }
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

    addEventListener("beforeunload", this.beforeUnload);

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChangedDebounced);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);
    bus.on(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
    bus.on(CHAT_ADD, this.onNewChat);
    bus.on(CHAT_EDITED, this.onEditChat);
    bus.on(CHAT_REDRAW, this.redrawItem);
    bus.on(CHAT_DELETED, this.onDeleteChat);
    bus.on(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.on(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);
    bus.on(USER_TYPING, this.onUserTyping);

    if (this.routeName == chat_list_name) {
      this.setTopTitle();
      this.chatStore.isShowSearch = true;
      this.chatStore.searchType = SEARCH_MODE_CHATS;
    }

    writingUsersTimerId = setInterval(()=>{
      for (const chatItem of this.items) {
        chatItem.writingUsers = filterOutOldWritingUsers(chatItem.writingUsers);
        if (chatItem.writingUsers.length == 0) {
          chatItem.usersWritingSubtitleInfo = null;
        } else {
          chatItem.usersWritingSubtitleInfo = buildWritingUsersSubtitleInfo(chatItem.writingUsers, this.$vuetify);
        }
      }
    }, 500);

    this.installOnFocus();
  },

  beforeUnmount() {
    this.saveLastVisibleElement();

    removeEventListener("beforeunload", this.beforeUnload);

    this.uninstallOnFocus();

    this.graphQlUserStatusUnsubscribe();
    this.uninstallScroller();

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChangedDebounced);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(UNREAD_MESSAGES_CHANGED, this.onChangeUnreadMessages);
    bus.off(CHAT_ADD, this.onNewChat);
    bus.off(CHAT_EDITED, this.onEditChat);
    bus.off(CHAT_REDRAW, this.redrawItem);
    bus.off(CHAT_DELETED, this.onDeleteChat);
    bus.off(CO_CHATTED_PARTICIPANT_CHANGED, this.onUserProfileChanged);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
    bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
    bus.off(VIDEO_CALL_SCREEN_SHARE_CHANGED, this.onVideoScreenShareChanged);
    bus.off(USER_TYPING, this.onUserTyping);

    clearInterval(writingUsersTimerId);

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
  font-weight 1000
}

.chat-normal {
  font-weight 400
}

.subtitle-thin {
  font-weight 300
}
</style>
