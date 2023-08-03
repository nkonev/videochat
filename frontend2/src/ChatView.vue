<template>

    <v-container :style="heightWithoutAppBar" fluid>
        <div class="my-messages-scroller" @scroll.passive="onScroll">
          <div class="first-element" style="min-height: 1px; background: #9cffa1"></div>
          <div v-for="item in items" :key="item.id" class="card mb-3" :id="getItemId(item.id)">
            <div class="row g-0">
              <div class="col">
                <img :src="item.owner.avatar" style="max-width: 64px; max-height: 64px">
              </div>
              <div class="col">
                <div class="card-body">
                  <h5 class="card-title">{{ item.text }}</h5>
                </div>
              </div>
            </div>
          </div>
          <div class="last-element" style="min-height: 1px; background: #c62828"></div>

        </div>

    </v-container>

</template>

<script>
    import axios from "axios";
    import infiniteScrollMixin, {directionTop, reduceToLength} from "@/mixins/infiniteScrollMixin";
    import heightMixin from "@/mixins/heightMixin";
    import searchString from "@/mixins/searchString";
    import bus, {PROFILE_SET, SEARCH_STRING_CHANGED} from "@/bus/bus";
    import {hasLength} from "@/utils";
    import debounce from "lodash/debounce";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";

    const PAGE_SIZE = 40;

    export default {
      mixins: [
        infiniteScrollMixin(),
        heightMixin(),
        searchString(),
      ],
      data() {
        return {
          startingFromItemIdTop: null,
          startingFromItemIdBottom: null,
        }
      },

      computed: {
        ...mapStores(useChatStore),
        chatId() {
          return this.$route.params.id
        },
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
        saveScroll(bottom) {
            this.preservedScroll = bottom ? this.getMaximumItemId() : this.getMinimumItemId();
            console.log("Saved scroll", this.preservedScroll);
        },
        initialDirection() {
          return directionTop
        },
        onFirstLoad() {
          this.scrollDown();
          this.loadedBottom = true;
        },
        async load() {
          if (!this.canDrawMessages()) {
              return Promise.resolve()
          }

          const startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
          return axios.get(`/api/chat/${this.chatId}/message`, {
              params: {
                startingFromItemId: startingFromItemId,
                size: PAGE_SIZE,
                reverse: this.isTopDirection(),
                searchString: this.searchString,
              },
            })
          .then((res) => {
            const items = res.data;
            console.log("Get items", items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom, "chosen", startingFromItemId);

            if (this.isTopDirection()) {
              this.items = this.items.concat(items);
            } else {
              this.items = items.reverse().concat(this.items);
            }

            if (items.length < PAGE_SIZE) {
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
          }).then(()=>{
            return this.$nextTick()
          })
        },

        bottomElementSelector() {
          return ".first-element"
        },
        topElementSelector() {
          return ".last-element"
        },
        getItemId(id) {
          return 'item-' + id
        },

        scrollDown() {
          this.$nextTick(() => {
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
        reloadItems() {
          this.reset();
          this.loadTop();
        },
        onSearchStringChanged() {
          this.reloadItems();
        },
        onProfileSet() {
          this.reloadItems();
        },

        canDrawMessages() {
          return !!this.chatStore.currentUser && hasLength(this.chatId)
        },
      },
      created() {
        this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
      },

      mounted() {
        this.initScroller();
        bus.on(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
        bus.on(PROFILE_SET, this.onProfileSet);
      },

      beforeUnmount() {
        this.destroyScroller();
        bus.off(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
        bus.off(PROFILE_SET, this.onProfileSet);
      }
    }
</script>

<style lang="stylus">
    .my-messages-scroller {
      height 100%
      overflow-y scroll !important
      display flex
      flex-direction column-reverse
    }

</style>
