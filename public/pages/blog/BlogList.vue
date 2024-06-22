<template>
  <v-container class="ma-0 pa-0 my-list-container" fluid>
    <div class="my-blog-scroller" id="blog-post-list">

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
                <v-card-title>
                  <a class="post-title-text" v-html="item.title" :href="getLink(item)"></a>
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

      <v-pagination v-model="page" @update:modelValue="onClickPage" :length="pagesCount">
      </v-pagination>
    </div>

  </v-container>
</template>

<script>
import {getHumanReadableDate, hasLength, getLoginColoredStyle, SEARCH_MODE_POSTS, PAGE_PARAM} from "#root/common/utils";
import {path_prefix, blog_post, blogIdPrefix, profile, blog} from "#root/common/router/routes";
import {usePageContext} from "#root/renderer/usePageContext.js";
import debounce from "lodash/debounce.js";
import bus, {SEARCH_STRING_CHANGED} from "#root/common/bus.js";

export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
        pageContext
    }
  },
  data() {
      return this.pageContext.data;
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
      let actualPage = e--;

      const url = new URL(window.location.href);
      url.searchParams.set(PAGE_PARAM, actualPage);

      window.location.href = url.toString();
    },
    onSearchStringChanged(v) {
      const url = new URL(window.location.href);
      url.searchParams.delete(PAGE_PARAM);
      if (v) {
          url.searchParams.set(SEARCH_MODE_POSTS, v);
      } else {
          url.searchParams.delete(SEARCH_MODE_POSTS);
      }
      window.location.href = url.toString();
    },
  },
  computed: {
  },
  created() {
      this.onSearchStringChanged = debounce(this.onSearchStringChanged, 700, {leading:false, trailing:true})
  },
  mounted() {
      bus.on(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
  },
  beforeUnmount() {
      bus.off(SEARCH_STRING_CHANGED, this.onSearchStringChanged);
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
