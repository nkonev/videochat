<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="user-list-items" class="my-user-scroller" @scroll.passive="onScroll">
            <div class="user-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                class="list-item-prepend-spacer-16 pb-2 user-item-root"
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
                        <span class="user-name" v-html="getUserNameOverride(item)"></span>
                    </v-list-item-title>
                    <v-list-item-subtitle>
                      <v-chip
                        density="comfortable"
                        v-if="item.oauth2Identifiers.vkontakteId"
                        class="mr-1 c-btn-vk cursor-pointer"
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
                        class="mr-1 c-btn-fb cursor-pointer"
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
                        class="mr-1 c-btn-google cursor-pointer"
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
                        class="mr-1 c-btn-keycloak cursor-pointer"
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

                      <template v-if="item.additionalData">
                      <v-chip v-for="(role, index) in item.additionalData.roles"
                        density="comfortable"
                        class="mr-1 cursor-pointer"
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
      >
      </UserListContextMenu>
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
    CLOSE_SIMPLE_MODAL,
    LOGGED_OUT, OPEN_SIMPLE_MODAL,
    PROFILE_SET,
    SEARCH_STRING_CHANGED
} from "@/bus/bus";
import {searchString, goToPreservingQuery, SEARCH_MODE_USERS} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
    deepCopy, findIndex,
    hasLength, isSetEqual, replaceInArray,
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

const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'UserList';

export default {
  components: {
      UserListContextMenu
  },
  mixins: [
    infiniteScrollMixin(scrollerName),
    hashMixin(),
    heightMixin(),
    searchString(SEARCH_MODE_USERS),
    graphqlSubscriptionMixin('userAccountEvents'),
    userStatusMixin('userStatusInUserList'),
  ],
  data() {
    return {
        markInstance: null,
    }
  },
  computed: {
    ...mapStores(useChatStore),
    showProgress() {
      return this.chatStore.progressCount > 0
    },
    itemIds() {
      return this.items.map(i => i.id);
    },
  },

  methods: {
    getUserNameOverride(item) {
      if (item.additionalData && (!item.additionalData.confirmed || item.additionalData.locked)) {
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
      this.items = this.items.slice(0, this.getReduceToLength());
      this.startingFromItemIdBottom = this.getMaximumItemId();
    },
    reduceTop() {
      this.items = this.items.slice(-this.getReduceToLength());
      this.startingFromItemIdTop = this.getMinimumItemId();
    },
    saveScroll(top) {
      this.preservedScroll = top ? this.getMinimumItemId() : this.getMaximumItemId();
      console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    initialDirection() {
      return directionBottom
    },
    async scrollTop() {
      return await this.$nextTick(() => {
          this.scrollerDiv.scrollTop = 0;
      });
    },
    async onFirstLoad(loadedResult) {
      await this.doScrollOnFirstLoad(userIdHashPrefix);
      if (loadedResult === true) {
        removeTopUserPosition();
      }
    },
    async doDefaultScroll() {
      this.loadedTop = true;
      await this.scrollTop();
    },
    getPositionFromStore() {
      return getTopUserPosition()
    },

    async load() {
      if (!this.canDrawUsers()) {
        return Promise.resolve()
      }

      this.chatStore.incrementProgressCount();

      const { startingFromItemId, hasHash } = this.prepareHashesForLoad();

      return axios.post(`/api/aaa/user/search`, {
          startingFromItemId: startingFromItemId,
          size: PAGE_SIZE,
          reverse: this.isTopDirection(),
          searchString: this.searchString,
          hasHash: hasHash,
        })
        .then((res) => {
          const items = res.data;
          console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom);
          items.forEach((item) => {
            this.transformItem(item);
          });

          if (this.isTopDirection()) {
              replaceOrPrepend(this.items, items);
          } else {
              replaceOrAppend(this.items, items);
          }

          if (items.length < PAGE_SIZE) {
            if (this.isTopDirection()) {
              this.loadedTop = true;
            } else {
              this.loadedBottom = true;
            }
          }
          this.updateTopAndBottomIds();

          if (!this.isFirstLoad) {
            this.clearRouteHash()
          }

          this.graphQlUserStatusSubscribe();
          this.performMarking();
          return Promise.resolve(true)
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
      await this.setHashAndReloadItems();
    },

    saveLastVisibleElement() {
      console.log("saveLastVisibleElement", !this.isScrolledToTop())
      if (!this.isScrolledToTop()) {
          const elems = [...document.querySelectorAll(this.scrollerSelector() + " .user-item-root")].map((item) => {
              const visible = item.getBoundingClientRect().top > 0
              return {item, visible}
          });

          const visible = elems.filter((el) => el.visible);
          // console.log("visible", visible, "elems", elems);
          if (visible.length == 0) {
              console.warn("Unable to get top visible")
              return
          }
          const topVisible = visible[0].item

          const uid = this.getIdFromRouteHash(topVisible.id);
          console.log("Found bottomUser", topVisible, "userId", uid);

          setTopUserPosition(uid)
      } else {
          console.log("Skipped saved topVisible because we are already scrolled to the bottom ")
      }
    },
    beforeUnload() {
      this.saveLastVisibleElement();
    },

    onLoggedOut() {
      this.reset();
      this.graphQlUserStatusUnsubscribe();
      this.graphQlUnsubscribe();
      this.beforeUnload();
    },

    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
    openUser(item){
          goToPreservingQuery(this.$route, this.$router, { name: profile_name, params: { id: item.id}})
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
      const userIds = this.getUserIdsSubscribeTo();
      return `
                subscription {
                  userAccountEvents(userIds:[${userIds}]) {
                    userAccountEvent {
                      ... on UserAccountExtendedDto {
                        id
                        login
                        avatar
                        avatarBig
                        shortInfo
                        lastLoginDateTime
                        oauth2Identifiers {
                          facebookId
                          vkontakteId
                          googleId
                          keycloakId
                        }
                        additionalData {
                          enabled
                          expired
                          locked
                          confirmed
                          roles
                        }
                        canLock
                        canDelete
                        canChangeRole
                        canConfirm
                      }
                      ... on UserAccountDto {
                        id
                        login
                        avatar
                        avatarBig
                        shortInfo
                        lastLoginDateTime
                        oauth2Identifiers {
                          facebookId
                          vkontakteId
                          googleId
                          keycloakId
                        }
                      }
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
        this.changeItem(tmp);
        this.performMarking();
      } else if (d.eventType === 'user_account_deleted') {
          this.removeItem(d.userAccountEvent);
      }
    },

    // does should change items list (new item added to visible part or not for example)
    hasItem(item) {
      let idxOf = findIndex(this.items, item);
      return idxOf !== -1;
    },

    changeItem(dto) {
      console.log("Replacing item", dto);
      replaceInArray(this.items, dto);
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

    onShowContextMenu(e, menuableItem) {
      this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
    },
    unlockUser(user) {
        axios.post('/api/aaa/user/lock', {userId: user.id, lock: false});
    },
    lockUser(user) {
        axios.post('/api/aaa/user/lock', {userId: user.id, lock: true});
    },
    tetATet(user) {
        axios.put(`/api/chat/tet-a-tet/${user.id}`).then(response => {
            this.$router.push(({ name: chat_name, params: { id: response.data.id}}));
        })
    },
    unconfirmUser(user) {
        axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: false});
    },
    confirmUser(user) {
        axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: true});
    },
    deleteUser(user) {
        bus.emit(OPEN_SIMPLE_MODAL, {
            buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
            title: this.$vuetify.locale.t('$vuetify.delete_user_title', user.id),
            text: this.$vuetify.locale.t('$vuetify.delete_user_text', user.login),
            actionFunction: (that) => {
                that.loading = true;
                axios.delete('/api/aaa/user', { params: {
                        userId: user.id
                    }}).then(() => {
                        bus.emit(CLOSE_SIMPLE_MODAL);
                    }).finally(()=>{
                    that.loading = false;
                })
            }
        });
    },
  },
  created() {
    this.onSearchStringChanged = debounce(this.onSearchStringChanged, 700, {leading:false, trailing:true})
  },
  watch: {
      '$vuetify.locale.current': {
          handler: function (newValue, oldValue) {
              this.setTopTitle();
          },
      },
      itemIds: function(newValue, oldValue) {
        if (newValue.length == 0) {
          this.graphQlUnsubscribe();
        } else {
          if (!isSetEqual(oldValue, newValue)) {
            this.graphQlSubscribe();
          }
        }
      },
      '$route': {
          handler: async function (newValue, oldValue) {

              // reaction on setting hash
              if (hasLength(newValue.hash)) {
                  console.log("Changed route hash, going to scroll", newValue.hash)
                  await this.scrollToOrLoad(newValue.hash);
                  return
              }
          }
      }
  },
  async mounted() {
    this.markInstance = new Mark("div#user-list-items .user-name");
    this.setTopTitle();
    this.chatStore.isShowSearch = true;
    this.chatStore.searchType = SEARCH_MODE_USERS;

    if (this.canDrawUsers()) {
      await this.onProfileSet();
    }

    addEventListener("beforeunload", this.beforeUnload);

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_USERS, this.onSearchStringChanged);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);

  },

  beforeUnmount() {
    // an analogue of watch(effectively(chatId)) in MessageList.vue
    // used when the user presses Start in the RightPanel
    this.saveLastVisibleElement();

    this.markInstance.unmark();
    this.markInstance = null;
    removeEventListener("beforeunload", this.beforeUnload);

    this.uninstallScroller();
    this.graphQlUserStatusUnsubscribe();
    this.graphQlUnsubscribe();

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_USERS, this.onSearchStringChanged);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);

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

.cursor-pointer {
  cursor pointer
}

</style>
