<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="user-list-items" class="my-user-scroller" @scroll.passive="onScroll">
            <div class="user-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                class="list-item-prepend-spacer pb-2 user-item-root"
                @contextmenu.stop="onShowContextMenu($event, item)"
                @click.prevent="openUser(item)"
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
                        <span class="user-name" :style="getLoginColoredStyle(item)" v-html="getUserNameOverride(item)"></span>
                    </v-list-item-title>
                    <v-list-item-subtitle>
                      <v-chip
                        density="comfortable"
                        v-if="item.oauth2Identifiers.vkontakteId"
                        class="mr-1 c-btn-vk"
                        text-color="white"
                      >
                        <template v-slot:prepend>
                          <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}"></font-awesome-icon>
                        </template>
                        <template v-slot:default>
                          <span class="ml-1">
                            Vkontakte
                          </span>
                        </template>
                      </v-chip>
                      <v-chip
                        density="comfortable"
                        v-if="item.oauth2Identifiers.facebookId"
                        class="mr-1 c-btn-fb"
                        text-color="white"
                      >
                        <template v-slot:prepend>
                          <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}"></font-awesome-icon>
                        </template>
                        <template v-slot:default>
                          <span class="ml-1">
                            Facebook
                          </span>
                        </template>
                      </v-chip>
                      <v-chip
                        density="comfortable"
                        v-if="item.oauth2Identifiers.googleId"
                        class="mr-1 c-btn-google"
                        text-color="white"
                      >
                        <template v-slot:prepend>
                          <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}"></font-awesome-icon>
                        </template>
                        <template v-slot:default>
                          <span class="ml-1">
                            Google
                          </span>
                        </template>
                      </v-chip>
                      <v-chip
                        density="comfortable"
                        v-if="item.oauth2Identifiers.keycloakId"
                        class="mr-1 c-btn-keycloak"
                        text-color="white"
                      >
                        <template v-slot:prepend>
                          <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'key'}"></font-awesome-icon>
                        </template>
                        <template v-slot:default>
                          <span class="ml-1">
                            Keycloak
                          </span>
                        </template>
                      </v-chip>
                      <v-chip
                        density="comfortable"
                        v-if="item.ldap"
                        class="mr-1 c-btn-database"
                        text-color="white"
                      >
                        <template v-slot:prepend>
                            <font-awesome-icon :icon="{ prefix: 'fas', iconName: 'database'}"></font-awesome-icon>
                        </template>
                        <template v-slot:default>
                          <span class="ml-1">
                            Ldap
                          </span>
                        </template>
                      </v-chip>

                      <template v-if="item.additionalData">
                      <v-chip v-for="(role, index) in item.additionalData.roles"
                        density="comfortable"
                        class="mr-1"
                        text-color="white"
                      >
                        <template v-slot:default>
                          <span>
                            {{role}}
                          </span>
                        </template>
                      </v-chip>
                      </template>

                    </v-list-item-subtitle>
                </template>
                <template v-slot:append v-if="!isMobile()">
                    <v-list-item-action>
                        <v-btn variant="flat" icon @click.stop.prevent="tetATet(item)" :title="$vuetify.locale.t('$vuetify.user_open_chat')"><v-icon size="large">mdi-message-text-outline</v-icon></v-btn>
                    </v-list-item-action>
                </template>

            </v-list-item>
            <template v-if="items.length == 0 && !showProgress">
              <v-sheet class="mx-2">{{$vuetify.locale.t('$vuetify.users_not_found')}}</v-sheet>
            </template>
            <div class="user-last-element" style="min-height: 1px; background: white"></div>
      </v-list>
      <UserListContextMenu
          ref="contextMenuRef"
          @tetATet="this.tetATet"
          @unlockUser="this.unlockUser"
          @lockUser="this.lockUser"
          @unconfirmUser="this.unconfirmUser"
          @confirmUser="this.confirmUser"
          @deleteUser="this.deleteUser"
          @changeRole="this.changeRole"
          @removeSessions="this.removeSessions"
          @enableUser="this.enableUser"
          @disableUser="this.disableUser"
          @setPassword="this.setPassword"
      >
      </UserListContextMenu>
      <UserRoleModal/>
  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {
    directionBottom,
} from "@/mixins/infiniteScrollMixin";
import {chat_name, profile, profile_name, userIdHashPrefix, userIdPrefix} from "@/router/routes";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import heightMixin from "@/mixins/heightMixin";
import bus, {
  CHANGE_ROLE_DIALOG,
  CLOSE_SIMPLE_MODAL,
  LOGGED_OUT, OPEN_SET_PASSWORD_MODAL, OPEN_SIMPLE_MODAL,
  PROFILE_SET, REFRESH_ON_WEBSOCKET_RESTORED,
  SEARCH_STRING_CHANGED
} from "@/bus/bus";
import {searchString, SEARCH_MODE_USERS} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
  deepCopy, findIndex, getExtendedUserFragment, getLoginColoredStyle,
  hasLength, isSetEqual, isStrippedUserLogin, isUserHash, replaceInArray,
  replaceOrAppend,
  replaceOrPrepend,
  setTitle
} from "@/utils";
import Mark from "mark.js";
import userStatusMixin from "@/mixins/userStatusMixin";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import hashMixin from "@/mixins/hashMixin";
import {
    getTopUserPosition,
    removeTopUserPosition,
    setTopUserPosition,
} from "@/store/localStore";
import UserListContextMenu from "@/UserListContextMenu.vue";
import UserRoleModal from "@/UserRoleModal.vue";
import onFocusMixin from "@/mixins/onFocusMixin.js";

const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'UserList';

export default {
  components: {
      UserListContextMenu,
      UserRoleModal,
  },
  mixins: [
    infiniteScrollMixin(scrollerName),
    hashMixin(),
    heightMixin(),
    searchString(SEARCH_MODE_USERS),
    userStatusMixin('userStatusInUserList'), // another subscription
    onFocusMixin(),
  ],
  data() {
    return {
        markInstance: null,
        userEventsSubscription: null,
    }
  },
  computed: {
    ...mapStores(useChatStore),
    showProgress() {
      return this.chatStore.progressCount > 0
    },
    itemIds() {
      return this.getUserIdsSubscribeTo();
    },
  },

  methods: {
    getLoginColoredStyle,
    getUserNameOverride(item) {
      if (isStrippedUserLogin(item)) {
        return "<s>" + this.getUserName(item) + "</s>"
      } else {
        return this.getUserName(item)
      }
    },
    hasLength,
    getMaxItemsLength() {
        return 240
    },
    getReduceToLength() {
        return 80 // in case numeric pages, should complement with getMaxItemsLength() and PAGE_SIZE
    },
    reduceBottom() {
      this.items = this.items.slice(0, this.getReduceToLength()); // remove last from array, retain first N - reduce bottom on the page
      this.startingFromItemIdBottom = this.findBottomElementId();
    },
    reduceTop() {
      this.items = this.items.slice(-this.getReduceToLength()); // retain last n
      this.startingFromItemIdTop = this.findTopElementId();
    },
    enableHashInRoute() {
      return false
    },
    convertLoadedFromStoreHash(obj) {
      return userIdHashPrefix + obj
    },
    extractIdFromElementForStoring(element) {
      return this.getIdFromRouteHash(element.id)
    },
    saveScroll(top) {
      this.preservedScroll = top ? this.findTopElementId() : this.findBottomElementId();
      console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    initialDirection() {
      return directionBottom
    },
    async scrollTop() {
      removeTopUserPosition();
      return await this.$nextTick(() => {
          this.scrollerDiv.scrollTop = 0;
      });
    },
    async onFirstLoad(loadedResult) {
      await this.doScrollOnFirstLoad();
      if (loadedResult === true) {
        removeTopUserPosition();
      }
    },
    async doDefaultScroll() {
      await this.scrollTop();
    },
    getPositionFromStore() {
      return getTopUserPosition()
    },
    async fetchItems(startingFromItemId, reverse, includeStartingFrom) {
      const res = await axios.get(`/api/aaa/user/search`, {
        params: {
          startingFromItemId: startingFromItemId,
          includeStartingFrom: !!includeStartingFrom,
          size: PAGE_SIZE,
          reverse: reverse,
          searchString: this.searchString,
        }
      }, {
        signal: this.requestAbortController.signal
      })
      const items = res.data.items;
      console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom);
      items.forEach((item) => {
        this.transformItem(item);
      });

      return items
    },
    async load() {
      if (!this.canDrawUsers()) {
        return Promise.resolve()
      }

      this.chatStore.incrementProgressCount();

      const { startingFromItemId, hasHash } = this.prepareHashesForRequest();

      try {
        let items = await this.fetchItems(startingFromItemId, this.isTopDirection());
        if (hasHash) {
          const portion = await this.fetchItems(startingFromItemId, !this.isTopDirection(), true);
          items = portion.reverse().concat(items);
        }

        if (this.isTopDirection()) {
          replaceOrPrepend(this.items, items);
        } else {
          replaceOrAppend(this.items, items);
        }

        this.updateTopAndBottomIds();

        if (!this.isFirstLoad) {
          await this.clearRouteHash()
        }

        this.graphQlUserStatusSubscribe();
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
      return ".user-last-element"
    },
    topElementSelector() {
      return ".user-first-element"
    },

    getItemId(id) {
      return userIdPrefix + id
    },

    scrollerSelector() {
        return ".my-user-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.startingFromItemIdTop = null;
      this.startingFromItemIdBottom = null;
    },

    async onSearchStringChanged() {
      // Fixes excess delayed (because of debounce) reloading of items when (copied from ChatList.vue)
      if (this.isReady()) {
        await this.reloadItems();
      }
    },
    async onProfileSet() {
      await this.initializeHashVariablesAndReloadItems();
      this.userEventsSubscription.graphQlSubscribe();
    },
    conditionToSaveLastVisible() {
        return !this.isScrolledToTop();
    },
    itemSelector() {
        return '.user-item-root'
    },
    setPositionToStore(userId) {
        setTopUserPosition(userId)
    },
    beforeUnload() {
      this.saveLastVisibleElement();
    },

    onLoggedOut() {
      this.reset();
      this.graphQlUserStatusUnsubscribe();
      this.userEventsSubscription.graphQlUnsubscribe();
      this.beforeUnload();
    },

    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
    openUser(item){
        this.$router.push({ name: profile_name, params: { id: item.id}})
    },
    getLink(item) {
          return profile + "/" + item.id
    },
    setTopTitle() {
        this.chatStore.title = this.$vuetify.locale.t('$vuetify.users');
        setTitle(this.$vuetify.locale.t('$vuetify.users'));
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
    isScrolledToBottom() {
        if (this.scrollerDiv) {
            return Math.abs(this.scrollerDiv.scrollHeight - this.scrollerDiv.scrollTop - this.scrollerDiv.clientHeight) < SCROLLING_THRESHHOLD
        } else {
            return false
        }
    },
    updateTopAndBottomIds() {
      this.startingFromItemIdTop = this.findTopElementId();
      this.startingFromItemIdBottom = this.findBottomElementId();
    },

    getUserIdsSubscribeTo() {
        return this.items.map(item => item.id);
    },
    onUserStatusChanged(dtos) {
          if (dtos) {
              this.items.forEach(item => {
                  dtos.forEach(dtoItem => {
                      if (dtoItem.online !== null && item.id == dtoItem.userId) {
                          item.online = dtoItem.online;
                      }
                      if (dtoItem.isInVideo !== null && item.id == dtoItem.userId) {
                          item.isInVideo = dtoItem.isInVideo;
                      }

                  })
              })
          }
    },

    getGraphQlSubscriptionQuery() {
      return `
                subscription {
                  userAccountEvents {
                    userAccountEvent {
                      ${getExtendedUserFragment()},
                      ... on UserDeletedDto {
                        id
                      }
                    }
                    eventType
                  }
                }
            `
    },
    onNextSubscriptionElement(e) {
      const d = e.data?.userAccountEvents;
      if (d.eventType === 'user_account_changed') {
        const tmp = deepCopy(d.userAccountEvent);
        this.transformItem(tmp);
        this.onEditUser(tmp);
      } else if (d.eventType === 'user_account_deleted') {
          this.onDeleteUser(d.userAccountEvent);
      } else if (d.eventType === 'user_account_created') {
          const tmp = deepCopy(d.userAccountEvent);
          this.transformItem(tmp);
          this.onNewUser(tmp);
      }
    },

    // does should change items list (new item added to visible part or not for example)
    hasItem(item) {
      let idxOf = findIndex(this.items, item);
      return idxOf !== -1;
    },
    addItem(dto) {
      console.log("Adding item", dto);
      this.transformItem(dto);
      this.items.unshift(dto);
      this.reduceListAfterAdd(false);
      this.updateTopAndBottomIds();
    },
    changeItem(dto) {
      console.log("Replacing item", dto);
      this.transformItem(dto);
      replaceInArray(this.items, dto);
    },
    removeItem(dto) {
      if (this.hasItem(dto)) {
          console.log("Removing item", dto);
          const idxToRemove = findIndex(this.items, dto);
          this.items.splice(idxToRemove, 1);
          this.updateTopAndBottomIds();
      } else {
          console.log("Item was not be removed", dto);
      }
    },

    onNewUser(dto) {
      const isScrolledToTop = this.isScrolledToTop();
      const emptySearchString = !hasLength(this.searchString);
      if (isScrolledToTop && emptySearchString) {
          this.addItem(dto);
          this.performMarking();
          this.scrollTop();
      } else if (isScrolledToTop) { // not empty searchString
          axios.post(`/api/aaa/user/filter`, {
              searchString: this.searchString,
              userId: dto.id
          }, {
            signal: this.requestAbortController.signal
          }).then(({data}) => {
              if (data.found) {
                  this.addItem(dto);
                  this.performMarking();
                  this.scrollTop();
              }
          })
      } else {
          console.log("Skipping", dto, isScrolledToTop, emptySearchString)
      }
    },
    onDeleteUser(dto) {
      this.removeItem(dto);
    },
    onEditUser(dto) {
      this.changeItem(dto);
      this.performMarking();
    },


    onShowContextMenu(e, menuableItem) {
      this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
    },
    unlockUser(user) {
        axios.post('/api/aaa/user/lock', {userId: user.id, lock: false}, {
          signal: this.requestAbortController.signal
        });
    },
    lockUser(user) {
        axios.post('/api/aaa/user/lock', {userId: user.id, lock: true}, {
          signal: this.requestAbortController.signal
        });
    },
    enableUser(user) {
      axios.post('/api/aaa/user/enable', {userId: user.id, enable: true}, {
        signal: this.requestAbortController.signal
      });
    },
    disableUser(user) {
      axios.post('/api/aaa/user/enable', {userId: user.id, enable: false}, {
        signal: this.requestAbortController.signal
      });
    },
    setPassword(user) {
      bus.emit(OPEN_SET_PASSWORD_MODAL, {userId: user.id, userName: user.login})
    },
    tetATet(user) {
        axios.put(`/api/chat/tet-a-tet/${user.id}`, null, {
          signal: this.requestAbortController.signal
        }).then(response => {
            this.$router.push(({ name: chat_name, params: { id: response.data.id}}));
        })
    },
    unconfirmUser(user) {
        axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: false}, {
          signal: this.requestAbortController.signal
        });
    },
    confirmUser(user) {
        axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: true}, {
          signal: this.requestAbortController.signal
        });
    },
    deleteUser(user) {
        bus.emit(OPEN_SIMPLE_MODAL, {
            buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
            title: this.$vuetify.locale.t('$vuetify.delete_user_title', user.id),
            text: this.$vuetify.locale.t('$vuetify.delete_user_text', user.login),
            actionFunction: (that) => {
                that.loading = true;
                axios.delete('/api/aaa/user', {
                      params: {
                          userId: user.id
                      },
                      signal: this.requestAbortController.signal
                    }).then(() => {
                        bus.emit(CLOSE_SIMPLE_MODAL);
                    }).finally(()=>{
                    that.loading = false;
                })
            }
        });
    },
    changeRole(user) {
        bus.emit(CHANGE_ROLE_DIALOG, user)
    },
    removeSessions(user) {
        axios.delete('/api/aaa/sessions', {
            params: {
                userId: user.id
            },
            signal: this.requestAbortController.signal
        });
    },
    onFocus() {
      if (this.chatStore.currentUser && this.items) {
        this.requestStatuses();

        if (this.isScrolledToTop()) {
          const topNElements = this.items.slice(0, PAGE_SIZE);
          axios.post(`/api/aaa/user/fresh`, topNElements, {
            params: {
              size: PAGE_SIZE,
              searchString: this.searchString,
            },
            signal: this.requestAbortController.signal
          }).then((res)=>{
            if (!res.data.ok) {
              console.log("Need to update users");
              this.reloadItems();
            } else {
              console.log("No need to update users");
            }
          })
        }
      }
    },
    onWsRestoredRefresh() {
      this.saveLastVisibleElement();
      this.doOnFocus();
    },
    requestStatuses() {
      this.$nextTick(()=>{
          const list = this.items.map(item => item.id);
          const joined = list.join(",");

          this.triggerUsesStatusesEvents(joined, this.requestAbortController.signal);
      })
    },
    findBottomElementId() {
      return this.items[this.items.length-1]?.id
    },
    findTopElementId() {
      return this.items[0]?.id
    },
    isAppropriateHash(hash) {
      return isUserHash(hash)
    },
  },
  created() {
    this.onSearchStringChanged = debounce(this.onSearchStringChanged, 700, {leading:false, trailing:true});
    this.onWsRestoredRefresh = debounce(this.onWsRestoredRefresh, 300, {leading:false, trailing:true});
  },
  watch: {
      '$vuetify.locale.current': {
          handler: function (newValue, oldValue) {
              this.setTopTitle();
          },
      },
      itemIds: function(newArr, oldArr) {
          if (oldArr.length !== 0 && newArr.length === 0) {
              this.graphQlUserStatusUnsubscribe();
          } else {
              if (!isSetEqual(oldArr, newArr)) {
                  this.graphQlUserStatusSubscribe();
              }
          }
      },
  },
  async mounted() {
    this.markInstance = new Mark("div#user-list-items .user-name");
    this.setTopTitle();
    this.chatStore.isShowSearch = true;
    this.chatStore.searchType = SEARCH_MODE_USERS;

    // create subscription object before ON_PROFILE_SET
    this.userEventsSubscription = graphqlSubscriptionMixin('userAccountEvents', this.getGraphQlSubscriptionQuery, this.setErrorSilent, this.onNextSubscriptionElement);

    if (this.canDrawUsers()) {
      await this.onProfileSet();
    }

    addEventListener("beforeunload", this.beforeUnload);

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_USERS, this.onSearchStringChanged);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);
    bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);

    this.installOnFocus();
  },

  beforeUnmount() {
    this.uninstallOnFocus();

    this.graphQlUserStatusUnsubscribe();
    this.userEventsSubscription.graphQlUnsubscribe();
    this.userEventsSubscription = null;

    // an analogue of watch(effectively(chatId)) in MessageList.vue
    // used when the user presses Start in the RightPanel
    this.saveLastVisibleElement();

    this.markInstance.unmark();
    this.markInstance = null;
    removeEventListener("beforeunload", this.beforeUnload);

    this.uninstallScroller();

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_USERS, this.onSearchStringChanged);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);

    setTitle(null);
    this.chatStore.title = null;
    this.chatStore.isShowSearch = false;
  }
}
</script>

<style lang="stylus">
.my-user-scroller {
  height 100%
  overflow-y scroll !important
  display flex
  flex-direction column
}

</style>

<style lang="stylus" scoped>
@import "itemAvatar.styl"
@import "oAuth2.styl"

</style>
