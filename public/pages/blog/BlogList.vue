<template>
  <v-container class="ma-0 pa-0 my-list-container" fluid>
        <template v-if="pageContext.data.items.length">
            <div class="my-blog-scroller" id="blog-post-list">

                <v-card
                    v-for="(item, index) in pageContext.data.items"
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
                                    <v-card-title>
                                        <a class="post-title-text" v-html="item.title" :href="getLink(item)" @click="onPostClick()"></a>
                                    </v-card-title>
                                </v-container>
                            </v-img>
                        </v-card>
                    </v-card-text>

                    <v-card-text class="post-text pb-0" v-html="item.preview">
                    </v-card-text>

                    <v-card-actions v-if="item?.owner != null">
                        <v-list-item class="px-0 ml-2">
                            <template v-slot:prepend v-if="hasLength(item?.owner?.avatar)">
                                <div class="item-avatar mr-3">
                                    <a :href="getProfileLink(item.owner)" class="user-link">
                                        <img :src="item?.owner?.avatar">
                                    </a>
                                </div>
                            </template>

                            <template v-slot:default>
                                <v-list-item-title><a :href="getProfileLink(item.owner)" class="nodecorated-link" :style="getLoginColoredStyle(item.owner, true)">{{ item?.owner?.login }}</a></v-list-item-title>
                                <v-list-item-subtitle>
                                    {{ getDate(item) }}
                                </v-list-item-subtitle>

                            </template>

                        </v-list-item>
                    </v-card-actions>
                </v-card>

                <v-divider/>
                <v-container class="ma-0 pa-0" fluid>
                    <v-pagination
                        v-model="pageContext.data.page"
                        @update:modelValue="onClickPage"
                        :length="pageContext.data.pagesCount"
                        v-if="shouldShowPagination()"
                        :total-visible="pageContext.data.pagesCount < 10 && !isMobile() ? 10 : undefined"
                        variant="elevated"
                        active-color="primary"
                        density="comfortable"
                    />
                </v-container>
            </div>
        </template>
        <div v-else>
            <h1>Posts not found</h1>
        </div>
  </v-container>
</template>

<script>
import Mark from "mark.js";
import {getHumanReadableDate, hasLength, getLoginColoredStyle, SEARCH_MODE_POSTS, PAGE_PARAM, PAGE_SIZE} from "#root/common/utils";
import {path_prefix, blog_post, blogIdPrefix, blogIdHashPrefix, profile} from "#root/common/router/routes";
import {usePageContext} from "#root/renderer/usePageContext.js";
import debounce from "lodash/debounce.js";
import bus, {SEARCH_STRING_CHANGED} from "#root/common/bus.js";
import { navigate } from 'vike/client/router';
const SCROLLING_THRESHHOLD = 200; // px
const KEY_TOP_BLOG = 'topBlog';

export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
        pageContext
    }
  },
  data() {
      return {
          markInstance: null,
      }
  },
  methods: {
    getLoginColoredStyle,
    hasLength,
    isMobile() {
        return this.pageContext.isMobile
    },
    getDate(item) {
      return getHumanReadableDate(item.createDateTime)
    },
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
    },
    getLink(item) {
        return path_prefix + blog_post + "/" + item.id
    },
    getItemId(id) {
      return blogIdPrefix + id
    },
    onClickPage(e) {
      this.loading = true; // false will be set with the new data from server

      let actualPage = e--;

      const url = new URL(window.location.href);
      url.searchParams.set(PAGE_PARAM, actualPage);

      navigate(url.pathname + url.search);
    },
    onSearchStringChanged(searchString) {
        this.loading = true; // false will be set with the new data from server

        const url = new URL(window.location.href);

        url.searchParams.delete(PAGE_PARAM);
        if (searchString) {
            url.searchParams.set(SEARCH_MODE_POSTS, searchString);
        } else {
            url.searchParams.delete(SEARCH_MODE_POSTS);
        }

        navigate(url.pathname + url.search);
    },

    shouldShowPagination() {
        return this.pageContext.data.count > PAGE_SIZE
    },
    performMarking() {
      this.$nextTick(() => {
          this.markInstance.unmark();
          if (hasLength(this.pageContext.data.searchStringFacade)) {
              this.markInstance.mark(this.pageContext.data.searchStringFacade);
          }
      })
    },
    scrollerSelector() {
        return ".my-blog-scroller"
    },
    isScrolledToTop() {
      // in-place getting because of server-side rendering on changing page makes "cached" scrollerDiv invalid
      const scrollerDiv = document.querySelector(this.scrollerSelector());
      if (scrollerDiv) {
          return Math.abs(scrollerDiv.scrollTop) < SCROLLING_THRESHHOLD
      } else {
          return false
      }
    },
    getIdFromRouteHash(hash) {
        if (!hash) {
            return null;
        }
        const str = hash.replace(/\D/g, '');
        return hasLength(str) ? str : null;
    },
    setTopBlogPosition(blogId) {
        localStorage.setItem(KEY_TOP_BLOG, JSON.stringify(blogId));
    },
    removeTopBlogPosition() {
        localStorage.removeItem(KEY_TOP_BLOG);
    },
    getTopBlogPosition() {
        return JSON.parse(localStorage.getItem(KEY_TOP_BLOG));
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
              console.warn("Unable to get desiredVisible")
              return
          }
          const desiredVisible = visible[0].item

          const bid = this.getIdFromRouteHash(desiredVisible.id);
          console.log("For storing to localstore found desiredVisible", desiredVisible, "blogId", bid);

          this.setTopBlogPosition(bid)
      } else {
          console.log("Skipped saved desiredVisible because we are already scrolled")
      }
    },
    async doScrollOnFirstLoad(prefix) {
        const loadedFromStoreHash = this.getTopBlogPosition();
        if (loadedFromStoreHash) {
            await this.scrollTo(prefix + loadedFromStoreHash);
        }
    },
    async scrollTo(newValue) {
      return await this.$nextTick(()=>{
          const el = document.querySelector(newValue);
          el?.scrollIntoView({behavior: 'instant', block: "start"});
          return el
      })
    },
    async onFirstLoad(loadedResult) {
      await this.doScrollOnFirstLoad(blogIdHashPrefix);
      if (loadedResult === true) {
          this.removeTopBlogPosition();
      }
    },
    onPostClick() {
        this.setLoadingAnimation()
    },
    setLoadingAnimation() {
        this.loading = true
    },
  },
  computed: {
      loading: {
          get() {
              return this.pageContext.data.loading
          },
          set(v) {
              this.pageContext.data.loading = v;
          }
      }
  },
  watch: {
    'pageContext.data.items': function(newUserValue, oldUserValue) {
        this.performMarking();
    },
  },
  created() {
      this.onSearchStringChanged = debounce(this.onSearchStringChanged, 700, {leading:false, trailing:true})
  },
  mounted() {
      this.markInstance = new Mark("div#blog-post-list");
      bus.on(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
      this.performMarking(); // for initial
      addEventListener("beforeunload", this.saveLastVisibleElement);
      this.onFirstLoad(true);
  },
  beforeUnmount() {
      this.markInstance.unmark();
      this.markInstance = null;
      bus.off(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
      this.saveLastVisibleElement();
      removeEventListener("beforeunload", this.saveLastVisibleElement);
  },
}
</script>

<style lang="stylus">
@import "../../common/styles/constants.styl"
@import "../../common/styles/itemAvatar.styl"

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
