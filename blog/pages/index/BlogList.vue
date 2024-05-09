<template>
  <v-container class="ma-0 pa-0" :style="heightWithoutAppBar" fluid>
    <div class="my-blog-scroller" id="blog-post-list">
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
import {blog_post, blog_post_name, blogIdPrefix, blogIdHashPrefix, profile} from "#root/renderer/router/routes";
import heightMixin from "#root/renderer/mixins/heightMixin";
import {isMobileBrowser} from "#root/renderer/utils.js";
import {usePageContext} from "../../renderer/usePageContext.js";

export default {
  setup() {
    const pageContext = usePageContext();

    // expose to template and other options API hooks
    return {
        pageContext
    }
  },
  mixins: [
      heightMixin(),
  ],
  data() {
      return this.pageContext.data;
  },
  methods: {
    hasLength,
    isMobile() {
        return isMobileBrowser()
    },
    getDate(item) {
      return getHumanReadableDate(item.createDateTime)
    },
    getProfileLink(user) {
      let url = profile + "/" + user.id;
      return url;
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
    getItemId(id) {
      return blogIdPrefix + id
    },
  },
  computed: {
  },
  created() {
  },
  async mounted() {
  },
  beforeUnmount() {
  },
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
