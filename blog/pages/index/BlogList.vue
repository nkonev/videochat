<template>
  <v-container class="ma-0 pa-0" :style="heightWithoutAppBar" fluid>
    <div class="my-blog-scroller" id="blog-post-list" @scroll.passive="onScroll">
      <div class="blog-first-element" style="min-height: 1px; background: white"></div>

      <v-card
        v-for="(item, index) in items"
        :key="item.id"
        :id="getItemId(item.id)"
        class="mb-2 mr-2 blog-item-root"
        :min-width="isMobile() ? 200 : 400"
        max-width="600"
      >
        <v-card-text class="pb-0">
          <v-card>
            <v-img
              class="text-white align-end"
              gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
              cover
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
              <template v-slot:prepend v-if="hasLength(item?.owner?.avatar)">
                  <div class="item-avatar pr-0 mr-3">
                      <a :href="getProfileLink(item.owner)" class="user-link">
                          <img :src="item?.owner?.avatar">
                      </a>
                  </div>
              </template>

              <template v-slot:default>
                  <v-list-item-title><a :href="getProfileLink(item.owner)" class="colored-link">{{ item?.owner?.login }}</a></v-list-item-title>
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
import {getHumanReadableDate, hasLength, replaceOrAppend, replaceOrPrepend, setTitle} from "#root/renderer/utils";
import axios from "axios";
import debounce from "lodash/debounce";
import Mark from "mark.js";
import {blog_post, blog_post_name, blogIdPrefix, blogIdHashPrefix, profile} from "#root/renderer/router/routes";
import infiniteScrollMixin, {directionBottom, directionTop} from "#root/renderer/mixins/infiniteScrollMixin";
// import {mapStores} from "pinia";
// import {useBlogStore} from "@/store/blogStore";
// TODO
// import {goToPreservingQuery, SEARCH_MODE_POSTS, searchString} from "@/mixins/searchString";
// import bus, {SEARCH_STRING_CHANGED} from "@/bus/bus"; // TODO
import heightMixin from "#root/renderer/mixins/heightMixin";
import hashMixin from "#root/renderer/mixins/hashMixin";
import {
    getTopBlogPosition,
    removeTopBlogPosition,
    setTopBlogPosition,
} from "#root/renderer/store/localStore";
import {isMobileBrowser} from "#root/renderer/utils.js";
import { getData } from '#root/renderer/useData';
import {usePageContext} from "../../renderer/usePageContext.js";


const PAGE_SIZE = 40;
const SCROLLING_THRESHHOLD = 200; // px

const scrollerName = 'BlogList';

export default {
  mixins: [
      heightMixin(),
      infiniteScrollMixin(scrollerName),
      hashMixin(),
      // searchString(SEARCH_MODE_POSTS), // TODO
  ],
  data() {
      return getData();
  },
  methods: {
    hasLength,
    isMobile() {
        return isMobileBrowser()
    },
    getMaxItemsLength() {
        return 240
    },
    getReduceToLength() {
        return 80 // in case numeric pages, should complement with getMaxItemsLength() and PAGE_SIZE
    },
    reduceBottom() {
      this.items = this.items.slice(0, this.getReduceToLength());
      this.startingFromItemIdBottom = this.getMaximumItemId();
    },
    reduceTop() {
      this.items = this.items.slice(-this.getReduceToLength());
      this.startingFromItemIdTop = this.getMinimumItemId();
    },
    initialDirection() {
          return directionBottom
    },
    saveScroll(top) {
        this.preservedScroll = top ? this.getMaximumItemId() : this.getMinimumItemId();
        console.log("Saved scroll", this.preservedScroll, "in ", scrollerName);
    },
    async scrollTop() {
        return await this.$nextTick(() => {
            this.scrollerDiv.scrollTop = 0;
        });
    },
    async onFirstLoad(loadedResult) {
      await this.doScrollOnFirstLoad(blogIdHashPrefix);
      if (loadedResult === true) {
          removeTopBlogPosition();
      }
    },
    async doDefaultScroll() {
      this.loadedTop = true;
      await this.scrollTop();
    },
    getPositionFromStore() {
      return getTopBlogPosition()
    },

    async load() {
        console.log("in load");
        if (!this.canDrawBlogs()) {
            return Promise.resolve()
        }

        if (this.items.length) {
            this.updateTopAndBottomIds();
            this.performMarking();
            return Promise.resolve()
        }

        // this.blogStore.incrementProgressCount(); // TODO
        const { startingFromItemId, hasHash } = this.prepareHashesForLoad();
        return axios.get(`/api/blog`, {
            params: {
                startingFromItemId: startingFromItemId,
                size: PAGE_SIZE,
                reverse: this.isTopDirection(),
                searchString: this.searchString,
                hasHash: hasHash,
            },
        })
            .then((res) => {
                const items = res.data;
                console.log("Get items in ", scrollerName, items, "page", this.startingFromItemIdTop, this.startingFromItemIdBottom);

                // replaceOrPrepend() and replaceOrAppend() for the situation when order has been changed on server,
                // e.g. some chat has been popped up on sever due to somebody updated it
                if (this.isTopDirection()) {
                    replaceOrPrepend(this.items, items);
                } else {
                    replaceOrAppend(this.items, items);
                }

                if (items.length < PAGE_SIZE) {
                    if (this.isTopDirection()) {
                        this.loadedTop = true;
                    } else {
                        this.loadedBottom = true;
                    }
                }
                this.updateTopAndBottomIds();

                if (!this.isFirstLoad) {
                    this.clearRouteHash()
                }

                this.performMarking();
                return Promise.resolve(true)
            }).finally(()=>{
                // this.blogStore.decrementProgressCount(); // TODO
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
        return blogIdPrefix + id
    },

    scrollerSelector() {
        return ".my-blog-scroller"
    },
    reset(skipResetting) {
      this.resetInfiniteScrollVars(skipResetting);

      this.startingFromItemIdTop = null;
      this.startingFromItemIdBottom = null;
    },

    getDate(item) {
      return getHumanReadableDate(item.createDateTime)
    },

    performMarking() {
      // TODO
      // this.$nextTick(() => {
      //   if (hasLength(this.searchString)) {
      //     this.markInstance.unmark();
      //     this.markInstance.mark(this.searchString);
      //   }
      // })
    },
    isScrolledToTop() {
      if (this.scrollerDiv) {
        return Math.abs(this.scrollerDiv.scrollTop) < SCROLLING_THRESHHOLD
      } else {
        return false
      }
    },
    updateTopAndBottomIds() {
      this.startingFromItemIdTop = this.getMaximumItemId();
      this.startingFromItemIdBottom = this.getMinimumItemId();
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
        // this.blogStore.title = this.$vuetify.locale.t('$vuetify.blogs'); // TODO
    },
    goToBlog(item) {
        // TODO
        // goToPreservingQuery(this.$route, this.$router, { name: blog_post_name, params: { id: item.id} })
    },
    getLink(item) {
        return blog_post + "/" + item.id
    },
    async start() {
        await this.setHashAndReloadItems(true);
    },

    saveLastVisibleElement() {
      console.log("saveLastVisibleElement", !this.isScrolledToTop())
      if (!this.isScrolledToTop()) {
          const elems = [...document.querySelectorAll(this.scrollerSelector() + " .blog-item-root")].map((item) => {
              const visible = item.getBoundingClientRect().top > 0
              return {item, visible}
          });

          const visible = elems.filter((el) => el.visible);
          // console.log("visible", visible, "elems", elems);
          if (visible.length == 0) {
              console.warn("Unable to get top visible")
              return
          }
          const topVisible = visible[0].item

          const bid = this.getIdFromRouteHash(topVisible.id);
          console.log("Found bottomPost", topVisible, "blogId", bid);

          setTopBlogPosition(bid)
      } else {
          console.log("Skipped saved topVisible because we are already scrolled to the bottom ")
      }
    },
    beforeUnload() {
      this.saveLastVisibleElement();
    },

  },
  computed: {
      // ...mapStores(useBlogStore), // TODO
  },
  created() {
      this.onSearchStringChanged = debounce(this.onSearchStringChanged, 700, {leading:false, trailing:true});

      console.log("AAAAAAAAAAAAAAAAAAAAAAa");
      const pc = usePageContext();
      console.log("pc a>", pc);

  },
  async mounted() {
      console.log("BBBBBBBBBBBBBBBB");
      const pc = usePageContext();
      console.log("pc b>", pc);


      // this.blogStore.isShowSearch = true; // TODO
    this.markInstance = new Mark("div#blog-post-list");
    this.setTopTitle();
    // this.blogStore.searchType = SEARCH_MODE_POSTS; // TODO

    if (this.canDrawBlogs()) {
        await this.start();
    }
    addEventListener("beforeunload", this.beforeUnload);

    // TODO
    // bus.on(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_POSTS, this.onSearchStringChanged);
  },
  beforeUnmount() {
    // this.blogStore.isShowSearch = false; // TODO

    // an analogue of watch(effectively(chatId)) in MessageList.vue
    // used when the user presses Start in the RightPanel
    this.saveLastVisibleElement();

    this.markInstance.unmark();
    this.markInstance = null;
    removeEventListener("beforeunload", this.beforeUnload);

    this.uninstallScroller();
    // TODO
    // bus.off(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_POSTS, this.onSearchStringChanged);
  },
  watch: {
    '$route': { // TODO check if working in vike
        handler: async function (newValue, oldValue) {

            // reaction on setting hash
            if (hasLength(newValue.hash)) {
                console.log("Changed route hash, going to scroll", newValue.hash)
                await this.scrollToOrLoad(newValue.hash);
                return
            }
        }
    }
  }
}
</script>

<style lang="stylus">
@import "../../renderer/styles/constants.styl"
@import "../../renderer/styles/itemAvatar.styl"

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

.blog-item-root {
  flex: 1 1 300px;
}
.user-link {
    height 100%
}

</style>
