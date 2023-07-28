<template>

    <v-container style="height: calc(100vh - 64px); background: darkgrey">
        <div class="my-scroller" @scroll.passive="onScroll">
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
    import infiniteScrollMixin from "@/mixins/infiniteScrollMixin";

    const PAGE_SIZE = 40;

    export default {
      mixins: [
        infiniteScrollMixin()
      ],
      data() {
        return {
          startingFromItemIdTop: null,
          startingFromItemIdBottom: null,
        }
      },

      computed: {
        chatId() {
          return this.$route.params.id
        }
      },

      methods: {
        async load() {
          const startingFromItemId = this.isTopDirection() ? this.startingFromItemIdTop : this.startingFromItemIdBottom;
          return axios.get(`/api/chat/${this.chatId}/message`, {
              params: {
                startingFromItemId: startingFromItemId,
                size: PAGE_SIZE,
                reverse: this.isTopDirection(),
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
      },

      created() {
      },

      mounted() {
        this.initScroller()
      },

      beforeUnmount() {
        this.destroyScroller()
      }
    }
</script>

<style lang="stylus">
    .my-scroller {
      height 100%
      overflow-y scroll !important
      display flex
      flex-direction column-reverse
    }

</style>
