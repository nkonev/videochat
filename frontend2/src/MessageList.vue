<template>

    <v-container :style="heightWithoutAppBar" fluid class="pa-0 ma-0">
        <div class="my-messages-scroller" @scroll.passive="onScroll">
          <div class="message-first-element" style="min-height: 1px; background: #9cffa1"></div>
          <MessageItem v-for="item in items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="chatId"
            :my="item.owner.id === chatStore.currentUser.id"
            :highlight="item.id == highlightMessageId"
          ></MessageItem>
          <div class="message-last-element" style="min-height: 1px; background: #c62828"></div>
        </div>

    </v-container>

</template>

<script>
    import axios from "axios";
    import infiniteScrollMixin, {directionTop, reduceToLength} from "@/mixins/infiniteScrollMixin";
    import heightMixin from "@/mixins/heightMixin";
    import {searchString, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
    import bus, {LOGGED_OUT, PROFILE_SET, SEARCH_STRING_CHANGED} from "@/bus/bus";
    import {hasLength} from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import MessageItem from "@/MessageItem.vue";
    import {messageIdHashPrefix, messageIdPrefix} from "@/router/routes";
    import router from "@/router";

    const PAGE_SIZE = 40;

    const scrollerName = 'MessageList';

    export default {
      mixins: [
        infiniteScrollMixin(scrollerName),
        heightMixin(),
        searchString(SEARCH_MODE_MESSAGES),
      ],
      data() {
        return {
          startingFromItemIdTop: null,
          startingFromItemIdBottom: null,

          hasInitialHash: false,
        }
      },

      computed: {
        ...mapStores(useChatStore),
        chatId() {
          return this.$route.params.id
        },
        highlightMessageId() {
            return this.getMessageId(this.$route.hash);
        },
      },

      components: {
          MessageItem
      },

      methods: {
        getMaximumItemId() {
          return Math.max(...this.items.map(it => it.id))
        },
        getMinimumItemId() {
          return Math.min(...this.items.map(it => it.id))
        },
        reduceBottom() {
          this.items = this.items.slice(-reduceToLength);
          this.startingFromItemIdBottom = this.getMaximumItemId();
        },
        reduceTop() {
          this.items = this.items.slice(0, reduceToLength);
          this.startingFromItemIdTop = this.getMinimumItemId();
        },
        saveScroll(top) {
            this.preservedScroll = top ? this.getMinimumItemId() : this.getMaximumItemId();
            console.log("Saved scroll", this.preservedScroll);
        },
        initialDirection() {
          return directionTop
        },
        onFirstLoad() {
            if (this.highlightMessageId) {
              this.scrollTo(messageIdHashPrefix + this.highlightMessageId);
            } else {
              this.loadedBottom = true;
              this.scrollDown();
            }
        },
        async load() {
          if (!this.canDrawMessages()) {
              return Promise.resolve()
          }

          const startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
          return axios.get(`/api/chat/${this.chatId}/message`, {
              params: {
                startingFromItemId: this.hasInitialHash ? this.highlightMessageId : startingFromItemId,
                size: PAGE_SIZE,
                reverse: this.isTopDirection(),
                searchString: this.searchString,
                hasHash: this.hasInitialHash
              },
            })
          .then((res) => {
            const items = res.data;
            console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom, "chosen", startingFromItemId);

            if (this.isTopDirection()) {
              this.items = this.items.concat(items);
            } else {
              this.items = items.reverse().concat(this.items);
            }

            if (!this.hasInitialHash && items.length < PAGE_SIZE) {
              if (this.isTopDirection()) {
                this.loadedTop = true;
              } else {
                this.loadedBottom = true;
              }
            } else {
              if (this.isTopDirection()) {
                this.startingFromItemIdTop = this.getMinimumItemId();
                if (!this.startingFromItemIdBottom) {
                  this.startingFromItemIdBottom = this.getMaximumItemId();
                }
              } else {
                this.startingFromItemIdBottom = this.getMaximumItemId();
                if (!this.startingFromItemIdTop) {
                  this.startingFromItemIdTop = this.getMinimumItemId();
                }
              }
            }

            this.hasInitialHash = false;
            if (!this.isFirstLoad) {
              this.clearRouteHash(this.$route)
            }
          }).then(()=>{
            return this.$nextTick()
          })
        },

        bottomElementSelector() {
          return ".message-first-element"
        },
        topElementSelector() {
          return ".message-last-element"
        },

        getItemId(id) {
          return messageIdPrefix + id
        },

        clearRouteHash(route) {
          console.log("Cleaning hash");
          this.$router.push({ hash: null, query: route.query })
        },
        scrollDown() {
          this.$nextTick(() => {
              if (this.scrollerDiv) {
                this.scrollerDiv.scrollTop = 0;
              }
          });
        },
        scrollerSelector() {
          return ".my-messages-scroller"
        },

        reset() {
          this.resetInfiniteScrollVars();

          this.startingFromItemIdTop = null;
          this.startingFromItemIdBottom = null;
        },
        async reloadItems() {
          this.reset();
          this.uninstallScroller();
          await this.loadTop();
          await this.$nextTick(() => {
            this.installScroller();
          })

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
        canDrawMessages() {
          return !!this.chatStore.currentUser && hasLength(this.chatId)
        },
        scrollTo(newValue) {
          this.$nextTick(()=>{
            const el = document.querySelector(newValue)
            el?.scrollIntoView({behavior: 'instant', block: "start"});
          })
        },
        installScroller() {
          this.timeout = setTimeout(()=>{
            this.$nextTick(()=>{
              this.initScroller();
              console.log("Scroller", scrollerName, "has been installed");
            })
          }, 1500);
        },
        uninstallScroller() {
          if (this.timeout) {
            clearTimeout(this.timeout);
          }
          this.destroyScroller();
          console.log("Scroller", scrollerName, "has been uninstalled");
        },
      },
      created() {
        this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
        this.hasInitialHash = hasLength(this.highlightMessageId);
      },

      watch: {
          chatId(newVal, oldVal) {
            //console.debug("Chat id has been changed", oldVal, "->", newVal);
            if (hasLength(newVal)) {
              this.reset();
              this.uninstallScroller();
              this.$nextTick(() => {
                this.installScroller();
              })
            }
          },
          '$route.hash': {
            handler: function (newValue, oldValue) {
              if (hasLength(newValue)) {
                console.log("Changed route hash, going to scroll")
                this.scrollTo(newValue);
              }
            }
          }
      },

      async mounted() {
        bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);

        this.chatStore.searchType = SEARCH_MODE_MESSAGES;

        await this.loadTop();
        this.installScroller();
      },

      beforeUnmount() {
        this.uninstallScroller();
        bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
      }
    }
</script>

<style lang="stylus">
    .my-messages-scroller {
      height 100%
      overflow-y scroll !important
      display flex
      flex-direction column-reverse
      background white
    }

</style>
