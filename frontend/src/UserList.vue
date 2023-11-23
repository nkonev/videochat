<template>

  <v-container :style="heightWithoutAppBar" fluid class="ma-0 pa-0">
      <v-list id="user-list-items" class="my-user-scroller" @scroll.passive="onScroll">
            <div class="user-first-element" style="min-height: 1px; background: white"></div>
            <v-list-item
                v-for="(item, index) in items"
                :key="item.id"
                :id="getItemId(item.id)"
                class="list-item-prepend-spacer-16 pb-2"
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
  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {
    directionBottom,
    directionTop,
} from "@/mixins/infiniteScrollMixin";
import {profile, profile_name} from "@/router/routes";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import heightMixin from "@/mixins/heightMixin";
import bus, {
    LOGGED_OUT,
    PROFILE_SET,
    SEARCH_STRING_CHANGED
} from "@/bus/bus";
import {searchString, goToPreserving, SEARCH_MODE_USERS} from "@/mixins/searchString";
import debounce from "lodash/debounce";
import {
  hasLength, isSetEqual, replaceInArray,
  replaceOrAppend,
  replaceOrPrepend,
  setTitle
} from "@/utils";
import Mark from "mark.js";
import userStatusMixin from "@/mixins/userStatusMixin";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";

const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'UserList';

export default {
  mixins: [
    infiniteScrollMixin(scrollerName),
    heightMixin(),
    searchString(SEARCH_MODE_USERS),
    graphqlSubscriptionMixin('userAccountEvents'),
    userStatusMixin('userStatusInUserList'),
  ],
  data() {
    return {
        pageTop: 0,
        pageBottom: 0,
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
      if (item.additionalData && !item.additionalData.confirmed) {
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
              .get(`/api/aaa/user/get-page`, {params: {id: id, size: PAGE_SIZE, searchString: this.searchString}})
              .then(({data}) => data.page) - 1; // as in load() -> axios.get().then()
          if (this.pageTop == -1) {
              this.pageTop = 0
          }
          console.log("Set page top", this.pageTop, "for id", id);
      } else {
          const id = this.findBottomElementId();
          //console.log("Going to get bottom page", aDirection, id);
          this.pageBottom = await axios
              .get(`/api/aaa/user/get-page`, {params: {id: id, size: PAGE_SIZE, searchString: this.searchString}})
              .then(({data}) => data.page);
          console.log("Set page bottom", this.pageBottom, "for id", id);
      }
    },
    async load() {
      if (!this.canDrawUsers()) {
        return Promise.resolve()
      }

      this.chatStore.incrementProgressCount();
      const page = this.isTopDirection() ? this.pageTop : this.pageBottom;
      return axios.post(`/api/aaa/user/search`, {
          page: page,
          size: PAGE_SIZE,
          searchString: this.searchString,
        })
        .then((res) => {
          const items = res.data.users;
          console.log("Get items in ", scrollerName, items, "page", page);
          items.forEach((item) => {
            this.transformItem(item);
          });

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
          this.graphQlUserStatusSubscribe();
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
      return ".user-last-element"
    },
    topElementSelector() {
      return ".user-first-element"
    },

    getItemId(id) {
      return 'user-item-' + id
    },

    scrollerSelector() {
        return ".my-user-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.pageTop = 0;
      this.pageBottom = 0;
    },

    async onSearchStringChanged() {
      // Fixes excess delayed (because of debounce) reloading of items when (copied from ChatList.vue)
      if (this.isReady()) {
        await this.reloadItems();
      }
    },
    async onProfileSet() {
      await this.reloadItems();
    },
    onLoggedOut() {
      this.reset();
      this.graphQlUserStatusUnsubscribe();
      this.graphQlUnsubscribe();
    },

    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
    openUser(item){
          goToPreserving(this.$route, this.$router, { name: profile_name, params: { id: item.id}})
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
                    }
                    eventType
                  }
                }
            `
    },
    onNextSubscriptionElement(e) {
      const d = e.data?.userAccountEvents;
      if (d.eventType === 'user_account_changed') {
        this.changeItem(d.userAccountEvent);
        this.performMarking();
      }
    },
    changeItem(dto) {
      console.log("Replacing item", dto);
      replaceInArray(this.items, dto);
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
  },
  async mounted() {
    this.markInstance = new Mark("div#user-list-items .user-name");
    this.setTopTitle();
    this.chatStore.isShowSearch = true;
    this.chatStore.searchType = SEARCH_MODE_USERS;

    if (this.canDrawUsers()) {
      await this.onProfileSet();
    }

    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_USERS, this.onSearchStringChanged);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);

  },

  beforeUnmount() {
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
