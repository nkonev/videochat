<template>

  <v-container :style="heightWithoutAppBar" fluid class="pa-0 ma-0">
    <div class="my-chat-scroller" @scroll.passive="onScroll">
      <div class="chat-first-element" style="min-height: 1px; background: #9cffa1"></div>
      <div v-for="item in items" :key="item.id" class="card mb-3" :id="getItemId(item.id)">
        <div class="row g-0">
          <div class="col">
            <img :src="item.avatar" style="max-width: 64px; max-height: 64px">
          </div>
          <div class="col">
            <div class="card-body">
              <h5 class="card-title" @click="goToChat(item.id)">{{ item.name }}</h5>
            </div>
          </div>
          <hr/>
        </div>
      </div>
      <div class="chat-last-element" style="min-height: 1px; background: #c62828"></div>

    </div>

  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {directionBottom, reduceToLength} from "@/mixins/infiniteScrollMixin";
import {chat_view_name} from "@/router/routes";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";
import heightMixin from "@/mixins/heightMixin";
import bus, {LOGGED_OUT, PROFILE_SET, SEARCH_STRING_CHANGED} from "@/bus/bus";
import {searchString, goToPreserving, SEARCH_MODE_CHATS} from "@/mixins/searchString";
import debounce from "lodash/debounce";

const PAGE_SIZE = 40;

const scrollerName = 'ChatList';

export default {
  mixins: [
    infiniteScrollMixin(scrollerName),
    heightMixin(),
    searchString(SEARCH_MODE_CHATS),
  ],
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
    reduceBottom() {
        this.items = this.items.slice(0, reduceToLength);
    },
    reduceTop() {
        this.items = this.items.slice(-reduceToLength);
    },
    findBottomElementId() {
        return this.items[this.items.length-1].id
    },
    findTopElementId() {
        return this.items[0].id
    },
    saveScroll(top) {
        this.preservedScroll = top ? this.findTopElementId() : this.findBottomElementId();
        console.log("Saved scroll", this.preservedScroll);
    },
    initialDirection() {
      return directionBottom
    },
    async onFirstLoad() {
      this.loadedTop = true;
      await this.scrollUp(); // we need it to prevent browser's scrolling
    },
    async onChangeDirection() {
      if (this.isTopDirection()) { // became
          const id = this.findTopElementId();
          this.pageTop = await axios
              .get(`/api/chat/get-page`, {params: {id: id, previous: true, size: PAGE_SIZE,}})
              .then(({data}) => data.page)
      } else {
          const id = this.findBottomElementId();
          this.pageBottom = await axios
              .get(`/api/chat/get-page`, {params: {id: id, previous: false, size: PAGE_SIZE,}})
              .then(({data}) => data.page)
      }
    },
    async load() {
      if (!this.canDrawChats()) {
        return Promise.resolve()
      }

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

          if (this.isTopDirection()) {
              this.items = items.concat(this.items);
          } else {
              this.items = this.items.concat(items);
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
        }).then(()=>{
          return this.$nextTick()
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
    async scrollUp() {
      return await this.$nextTick(() => {
        if (this.scrollerDiv) {
          this.scrollerDiv.scrollTop = 0;
        }
      });
    },

    scrollerSelector() {
        return ".my-chat-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.pageTop = 0;
      this.pageBottom = 0;
    },

    goToChat(id) {
        goToPreserving(this.$route, this.$router, { name: chat_view_name, params: { id: id}})
    },
    onSearchStringChanged() {
      this.reloadItems();
    },
    onProfileSet() {
      this.reloadItems();
    },
    onLoggedOut() {
      this.reset();
    },

    canDrawChats() {
      return !!this.chatStore.currentUser
    },
  },
  created() {
    this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
  },

  async mounted() {
    bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChanged);
    bus.on(PROFILE_SET, this.onProfileSet);
    bus.on(LOGGED_OUT, this.onLoggedOut);

    this.chatStore.searchType = SEARCH_MODE_CHATS;

    await this.initialLoad();
    this.installScroller();
  },

  beforeUnmount() {
    this.uninstallScroller();
    console.log("Scroller", scrollerName, "has been uninstalled");

    bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_CHATS, this.onSearchStringChanged);
    bus.off(PROFILE_SET, this.onProfileSet);
    bus.off(LOGGED_OUT, this.onLoggedOut);
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
