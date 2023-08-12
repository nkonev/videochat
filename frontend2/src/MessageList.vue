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
    import bus, {LOGGED_OUT, PROFILE_SET, SCROLL_DOWN, SEARCH_STRING_CHANGED} from "@/bus/bus";
    import {hasLength} from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import MessageItem from "@/MessageItem.vue";
    import {messageIdHashPrefix, messageIdPrefix} from "@/router/routes";
    import {getTopMessagePosition, removeTopMessagePosition, setTopMessagePosition} from "@/store/localStore";

    const PAGE_SIZE = 40;
    const SCROLLING_THRESHHOLD = 200; // px

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
          loadedHash: null,
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
        async onFirstLoad() {
            if (this.highlightMessageId) {
              await this.scrollTo(messageIdHashPrefix + this.highlightMessageId);
            } else if (this.loadedHash) {
              await this.scrollTo(messageIdHashPrefix + this.loadedHash);
            } else {
              await this.scrollDown(); // we need it to prevent browser's scrolling
              this.loadedBottom = true;
            }
            this.loadedHash = null;
            removeTopMessagePosition(this.chatId);
        },
        async load() {
          if (!this.canDrawMessages()) {
              return Promise.resolve()
          }

          let startingFromItemId;
          if (this.hasInitialHash) { // we need it here - it shouldn't be computable in order to be reset. The resetted value is need when we press "arrow down" after reload
            // how to check:
            // 1. click on hash
            // 2. reload page
            // 3. press "arrow down" (Scroll down)
            // 4. It is going to invoke this load method which will use cashed and reset hasInitialHash = false
            startingFromItemId = this.highlightMessageId
          } else if (this.loadedHash) {
            startingFromItemId = this.loadedHash
          } else {
            startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
          }

          let hasHash;
          if (this.hasInitialHash) {
            hasHash = this.hasInitialHash
          } else if (this.loadedHash) {
            hasHash = !!this.loadedHash
          } else {
            hasHash = false
          }

          return axios.get(`/api/chat/${this.chatId}/message`, {
              params: {
                startingFromItemId: startingFromItemId,
                size: PAGE_SIZE,
                reverse: this.isTopDirection(),
                searchString: this.searchString,
                hasHash: hasHash
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

            if (!hasHash && items.length < PAGE_SIZE) {
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
              this.clearRouteHash()
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

        clearRouteHash() {
          // console.log("Cleaning hash");
          this.$router.push({ hash: null, query: this.$route.query })
        },
        async scrollDown() {
          return await this.$nextTick(() => {
            this.scrollerDiv.scrollTop = 0;
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
        async scrollTo(newValue) {
          return await this.$nextTick(()=>{
            const el = document.querySelector(newValue)
            el?.scrollIntoView({behavior: 'instant', block: "start"});
          })
        },

        async onScrollDownButton() {
          this.clearRouteHash();
          return await this.reloadItems();
        },

        saveLastVisibleElement(chatId) {
          if (!this.isScrolledToBottom()) {
            const elems = [...document.querySelectorAll(this.scrollerSelector() + " .message-item-root")].map((item) => {
              const visible = item.getBoundingClientRect().top > 0
              return {item, visible}
            });

            const visible = elems.filter((el) => el.visible);
            // console.log("visible", visible, "elems", elems);
            if (visible.length == 0) {
              console.warn("Unable to get top visible")
              return
            }
            const topVisible = visible[visible.length - 1].item

            console.log("Found topVisible", topVisible, "in chat", chatId);

            setTopMessagePosition(chatId, this.getMessageId(topVisible.id))
          } else {
            console.log("Skipped saved topVisible because we are already scrolled to the bottom ")
          }
        },

        onScrollCallback() {
          this.chatStore.showScrollDown = !this.isScrolledToBottom();
        },
        isScrolledToBottom() {
          if (this.scrollerDiv) {
            return Math.abs(this.scrollerDiv.scrollTop) < SCROLLING_THRESHHOLD
          } else {
            return false
          }
        },
        beforeUnload() {
          this.saveLastVisibleElement(this.chatId);
        }
      },
      created() {
        this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
        this.hasInitialHash = hasLength(this.highlightMessageId);
        this.loadedHash = getTopMessagePosition(this.chatId);
      },

      watch: {
          async chatId(newVal, oldVal) {
            //console.debug("Chat id has been changed", oldVal, "->", newVal);
            this.saveLastVisibleElement(oldVal);

            if (hasLength(newVal)) {
              this.loadedHash = getTopMessagePosition(this.chatId); // reinit boolean flag (prepare for upcoming loading)
              await this.reloadItems();
            }
          },
          '$route.hash': {
            handler: async function (newValue, oldValue) {
              if (hasLength(newValue)) {
                console.log("Changed route hash, going to scroll")
                await this.scrollTo(newValue);
              }
            }
          }
      },

      async mounted() {
        addEventListener("beforeunload", this.beforeUnload);

        bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);
        bus.on(SCROLL_DOWN, this.onScrollDownButton);

        this.chatStore.searchType = SEARCH_MODE_MESSAGES;

        await this.initialLoad();
        this.installScroller();
      },

      beforeUnmount() {
        removeEventListener("beforeunload", this.beforeUnload);

        this.uninstallScroller();
        bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
        bus.off(SCROLL_DOWN, this.onScrollDownButton);

        this.chatStore.showScrollDown = false;
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
