<template xmlns="http://www.w3.org/1999/html">

  <v-container style="height: calc(100dvh - 64px); background: lightblue">
    <div class="my-chat-scroller" @scroll.passive="onScroll">
      <div class="first-element" style="min-height: 1px; background: #9cffa1"></div>
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
      <div class="last-element" style="min-height: 1px; background: #c62828"></div>

    </div>

  </v-container>

</template>

<script>
import axios from "axios";
import infiniteScrollMixin, {directionBottom, reduceToLength} from "@/mixins/infiniteScrollMixin";
import {chat_name} from "@/routes";
import {useChatStore} from "@/store/chatStore";
import {mapStores} from "pinia";

const PAGE_SIZE = 40;

export default {
  mixins: [
    infiniteScrollMixin()
  ],
  data() {
    return {
        pageTop: 0,
        pageBottom: 0,
    }
  },
  computed: {
    ...mapStores(useChatStore),
    userIsSet() {
      return !!this.chatStore.currentUser
    },
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
    saveScroll(bottom) {
        this.preservedScroll = bottom ? this.findBottomElementId() : this.findTopElementId();
        console.log("Saved scroll", this.preservedScroll);
    },
    initialDirection() {
      return directionBottom
    },
    onFirstLoad() {
      this.loadedTop = true;
      this.scrollUp();
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
      if (!this.userIsSet) {
        return Promise.resolve()
      }

      const page = this.isTopDirection() ? this.pageTop : this.pageBottom;
      return axios.get(`/api/chat`, {
        params: {
          page: page,
          size: PAGE_SIZE,
        },
      })
        .then((res) => {
          const items = res.data.data;
          console.log("Get items", items, "page", page);

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
      return ".last-element"
    },
    topElementSelector() {
      return ".first-element"
    },
    getItemId(id) {
      return 'item-' + id
    },

    scrollUp() {
      this.$nextTick(() => {
        this.scrollerDiv.scrollTop = 0;
      });
    },
    scrollerSelector() {
        return ".my-chat-scroller"
    },

    goToChat(id) {
        this.$router.push(({ name: chat_name, params: { id: id}}));
    }
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
.my-chat-scroller {
  height 100%
  overflow-y scroll !important
  display flex
  flex-direction column
}

</style>
