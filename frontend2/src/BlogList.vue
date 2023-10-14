<template>
  <v-container class="ma-0 pa-0" :style="heightWithoutAppBar" fluid>
    <div class="my-blog-scroller" id="blog-post-list" @scroll.passive="onScroll">
      <div class="blog-first-element" style="min-height: 1px; background: white"></div>

      <v-card
        v-for="(item, index) in items"
        :key="item.id"
        :id="getItemId(item.id)"
        class="mb-2 mr-2 myclass"
        :min-width="isMobile() ? 200 : 400"
        max-width="600"
      >
        <v-card-text class="pb-0">
          <v-card>
            <v-img
              class="text-white align-end"
              gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
              :height="isMobile() ? 200 : 300"
              :src="item.imageUrl"
            >
              <v-container class="post-title ma-0 pa-0">
                <v-card-title @click.prevent="goToBlog(item)">
                  <a class="post-title-text" v-html="item.title" :href="getLink(item)"></a>
                </v-card-title>
              </v-container>
            </v-img>
          </v-card>
        </v-card-text>

        <v-card-text class="post-text pb-0" v-html="item.preview">
        </v-card-text>

        <v-card-actions v-if="item?.owner != null">
          <v-list-item>
              <template v-slot:prepend>
                  <v-avatar :image="item?.owner?.avatar">
                  </v-avatar>
              </template>

              <template v-slot:default>
                  <v-list-item-title><a @click.prevent="onParticipantClick(item.owner)" :href="getProfileLink(item.owner)" class="colored-link">{{ item?.owner?.login }}</a></v-list-item-title>
                  <v-list-item-subtitle>
                      {{ getDate(item) }}
                  </v-list-item-subtitle>

              </template>

          </v-list-item>
        </v-card-actions>
      </v-card>
      <div class="blog-last-element" style="min-height: 1px; background: white"></div>

    </div>

  </v-container>
</template>

<script>
import {getHumanReadableDate, hasLength, replaceOrAppend, replaceOrPrepend, setTitle} from "@/utils";
import axios from "axios";
import debounce from "lodash/debounce";
import Mark from "mark.js";
import {blog_post, blog_post_name} from "@/router/blogRoutes";
import {profile, profile_name} from "@/router/routes";
import infiniteScrollMixin, {directionBottom, directionTop} from "@/mixins/infiniteScrollMixin";
import {mapStores} from "pinia";
import {useBlogStore} from "@/store/blogStore";
import {goToPreserving, SEARCH_MODE_POSTS, searchString} from "@/mixins/searchString";
import bus, {SEARCH_STRING_CHANGED} from "@/bus/bus";
import heightMixin from "@/mixins/heightMixin";

const PAGE_SIZE = 40;

const scrollerName = 'BlogList';

export default {
  mixins: [
      heightMixin(),
      infiniteScrollMixin(scrollerName),
      searchString(SEARCH_MODE_POSTS),
  ],
  data() {
    return {
      items: [],
      page: 0,
      markInstance: null,
    }
  },
  methods: {
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
                .get(`/api/blog/get-page`, {params: {id: id, size: PAGE_SIZE,}})
                .then(({data}) => data.page) - 1; // as in load() -> axios.get().then()
            if (this.pageTop == -1) {
                this.pageTop = 0
            }
            console.log("Set page top", this.pageTop, "for id", id);
        } else {
            const id = this.findBottomElementId();
            //console.log("Going to get bottom page", aDirection, id);
            this.pageBottom = await axios
                .get(`/api/blog/get-page`, {params: {id: id, size: PAGE_SIZE,}})
                .then(({data}) => data.page);
            console.log("Set page bottom", this.pageBottom, "for id", id);
        }
    },
    async load() {
        if (!this.canDrawBlogs()) {
            return Promise.resolve()
        }

        this.blogStore.incrementProgressCount();
        const page = this.isTopDirection() ? this.pageTop : this.pageBottom;
        return axios.get(`/api/blog`, {
            params: {
                page: page,
                size: PAGE_SIZE,
                searchString: this.searchString,
            },
        })
            .then((res) => {
                const items = res.data;
                console.log("Get items in ", scrollerName, items, "page", page);

                // replaceOrPrepend() and replaceOrAppend() for the situation when order has been changed on server,
                // e.g. some chat has been popped up on sever due to somebody updated it
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
                this.performMarking();
            }).finally(()=>{
                this.blogStore.decrementProgressCount();
                return this.$nextTick();
            })
    },
    canDrawBlogs() {
        return true
    },

    bottomElementSelector() {
        return ".blog-last-element"
    },
    topElementSelector() {
        return ".blog-first-element"
    },

    getItemId(id) {
        return 'blog-item-' + id
    },

    scrollerSelector() {
        return ".my-blog-scroller"
    },
    reset() {
      this.resetInfiniteScrollVars();

      this.pageTop = 0;
      this.pageBottom = 0;
    },

    getDate(item) {
      return getHumanReadableDate(item.createDateTime)
    },

    performMarking() {
      this.$nextTick(() => {
        if (hasLength(this.searchString)) {
          this.markInstance.unmark();
          this.markInstance.mark(this.searchString);
        }
      })
    },
    getBlogPostLink(item) {
      return {
        name: blog_post_name,
        params: {
          id: item.id
        }
      }
    },
    onParticipantClick(user) {
      const routeDto = { name: profile_name, params: { id: user.id }};
      this.$router.push(routeDto);
    },
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
    },
    async onSearchStringChanged() {
      // Fixes excess delayed (because of debounce) reloading of items when
      // 1. we've chosen __AVAILABLE_FOR_SEARCH
      // 2. then go to the Welcome
      // 3. without this change there will be excess delayed invocation
      // 4. but we've already destroyed this component, so it will be an error in the log
      if (this.isReady()) {
          await this.reloadItems();
      }
    },
    setTopTitle() {
        setTitle(this.$vuetify.locale.t('$vuetify.blogs'));
        this.blogStore.title = this.$vuetify.locale.t('$vuetify.blogs');
    },
    goToBlog(item) {
        goToPreserving(this.$route, this.$router, { name: blog_post_name, params: { id: item.id} })
    },
    getLink(item) {
        return blog_post + "/" + item.id
    },
    async start() {
        await this.reloadItems();
    },
  },
  computed: {
      ...mapStores(useBlogStore),
  },
  created() {
      this.onSearchStringChanged = debounce(this.onSearchStringChanged, 200, {leading:false, trailing:true})
  },
  async mounted() {
        this.blogStore.isShowSearch = true;
        this.markInstance = new Mark("div#blog-post-list");
        this.setTopTitle();
        this.blogStore.searchType = SEARCH_MODE_POSTS;

        if (this.canDrawBlogs()) {
            await this.start();
        }

        bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_POSTS, this.onSearchStringChanged);
  },
  beforeUnmount() {
        this.blogStore.isShowSearch = false;
        this.uninstallScroller();
        bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_POSTS, this.onSearchStringChanged);
  }
}
</script>

<style lang="stylus">
@import "constants.styl"

.my-blog-scroller {
  height 100%
  overflow-y scroll !important
  display flex
  flex-wrap wrap
  align-items start
}

.post-title {
  background rgba(0, 0, 0, 0.5);

  .post-title-text {
    cursor pointer
    color white
    text-decoration none
    word-break: break-word;
  }
}
.post-text {
    color $blackColor
}

.myclass {
  flex: 1 1 300px;
}
</style>
