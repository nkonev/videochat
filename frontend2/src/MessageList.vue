<template>
        <div class="ma-0 px-0 pt-0 pb-2 my-messages-scroller" @scroll.passive="onScroll">
          <div class="message-first-element" style="min-height: 1px; background: white"></div>
          <MessageItem v-for="item in items"
            :id="getItemId(item.id)"
            :key="item.id"
            :item="item"
            :chatId="chatId"
            :my="item.owner.id === chatStore.currentUser.id"
            :highlight="item.id == highlightMessageId"
            :canResend="chatDto.canResend"
            @deleteMessage="deleteMessage"
          ></MessageItem>
          <div class="message-last-element" style="min-height: 1px; background: white"></div>
        </div>

</template>

<script>
    import axios from "axios";
    import infiniteScrollMixin, {directionTop, reduceToLength} from "@/mixins/infiniteScrollMixin";
    import {searchString, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
    import bus, {
      CLOSE_SIMPLE_MODAL,
      LOGGED_OUT,
      OPEN_SIMPLE_MODAL,
      PROFILE_SET,
      SCROLL_DOWN,
      SEARCH_STRING_CHANGED
    } from "@/bus/bus";
    import {hasLength} from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import MessageItem from "@/MessageItem.vue";
    import {messageIdHashPrefix, messageIdPrefix} from "@/router/routes";
    import {getTopMessagePosition, removeTopMessagePosition, setTopMessagePosition} from "@/store/localStore";
    import Mark from "mark.js";

    const PAGE_SIZE = 40;
    const SCROLLING_THRESHHOLD = 200; // px

    const scrollerName = 'MessageList';

    export default {
      mixins: [
        infiniteScrollMixin(scrollerName),
        searchString(SEARCH_MODE_MESSAGES),
      ],
      props: ['chatDto'],
      data() {
        return {
          startingFromItemIdTop: null,
          startingFromItemIdBottom: null,

          hasInitialHash: false, // do we have hash in address line (message id)
          loadedHash: null, // keeps loaded message id from localstore the most top visible message - preserves scroll between page reload or switching between chats

          markInstance: null,
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
          return this.items.length ? Math.max(...this.items.map(it => it.id)) : null
        },
        getMinimumItemId() {
          return this.items.length ? Math.min(...this.items.map(it => it.id)) : null
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
            console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
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
            this.hasInitialHash = false;
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

            if (!this.isFirstLoad) {
              this.clearRouteHash()
            }
            this.performMarking();
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
          this.chatStore.showScrollDown = false;

          this.startingFromItemIdTop = null;
          this.startingFromItemIdBottom = null;
        },
        async onSearchStringChanged() {
          await this.reloadItems();
        },
        async onProfileSet() {
          this.hasInitialHash = hasLength(this.highlightMessageId);
          this.loadedHash = getTopMessagePosition(this.chatId);
          await this.reloadItems();
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
          await this.reloadItems();
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

            const mid = this.getMessageId(topVisible.id);
            console.log("Found topVisible", topVisible, "in chat", chatId, "messageId", mid);

            setTopMessagePosition(chatId, mid)
          } else {
            console.log("Skipped saved topVisible because we are already scrolled to the bottom ")
          }
        },
        beforeUnload() {
          this.saveLastVisibleElement(this.chatId);
        },
        performMarking() {
          this.$nextTick(() => {
            if (hasLength(this.searchString)) {
              this.markInstance.unmark();
              this.markInstance.mark(this.searchString);
            }
          })
        },

        deleteMessage(dto){
          bus.emit(OPEN_SIMPLE_MODAL, {
            buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
            title: this.$vuetify.locale.t('$vuetify.delete_message_title', dto.id),
            text:  this.$vuetify.locale.t('$vuetify.delete_message_text'),
            actionFunction: (that)=> {
              that.loading = true;
              axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
                .then(() => {
                  bus.emit(CLOSE_SIMPLE_MODAL);
                })
                .finally(()=>{
                  that.loading = false;
                })
            }
          });
        },

      },
      created() {
        this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
      },

      watch: {
          async chatId(newVal, oldVal) {
            console.debug("Chat id has been changed", oldVal, "->", newVal);
            this.saveLastVisibleElement(oldVal);
            if (hasLength(newVal)) {
              // this.loadedHash = getTopMessagePosition(newVal);
              // await this.reloadItems();
              await this.onProfileSet();
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
        this.markInstance = new Mark(this.scrollerSelector() + " .message-item-text");

        // we trigger actions on load if profile was set
        // else we rely on PROFILE_SET
        // should be before bus.on(PROFILE_SET, this.onProfileSet);
        if (this.canDrawMessages()) {
          await this.onProfileSet();
        }

        addEventListener("beforeunload", this.beforeUnload);

        bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);
        bus.on(SCROLL_DOWN, this.onScrollDownButton);

        this.chatStore.searchType = SEARCH_MODE_MESSAGES;
      },

      beforeUnmount() {
        this.markInstance.unmark();
        this.markInstance = null;
        removeEventListener("beforeunload", this.beforeUnload);

        this.reset();

        this.uninstallScroller();
        bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_MESSAGES, this.onSearchStringChanged);
        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
        bus.off(SCROLL_DOWN, this.onScrollDownButton);
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
