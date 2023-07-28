<template xmlns="http://www.w3.org/1999/html">

  <v-container style="height: calc(100vh - 64px); background: lightblue">
    <div class="my-scroller" @scroll.passive="onScroll">
      <div class="first-element" style="min-height: 1px; background: #9cffa1"></div>
      <div v-for="item in items" :key="item.id" class="card mb-3" :id="getItemId(item.id)">
        <div class="row g-0">
          <div class="col">
            <img :src="item.avatar" style="max-width: 64px; max-height: 64px">
          </div>
          <div class="col">
            <div class="card-body">
              <h5 class="card-title">{{ item.name }}</h5>
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
import infiniteScrollMixin from "@/mixins/infiniteScrollMixin";

const PAGE_SIZE = 40;

export default {
  mixins: [
    infiniteScrollMixin()
  ],
  data() {
    return {
      page: 0,
    }
  },

  methods: {
    onFirstLoad() {
      this.loadedTop = true;
    },
    async load() {
      return axios.get(`/api/chat`, {
        params: {
          page: this.page,
          size: PAGE_SIZE,
        },
      })
        .then((res) => {
          const items = res.data.data;
          console.log("Get items", items, "page", this.page);

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
            this.page += 1;
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
  flex-direction column
}

</style>
