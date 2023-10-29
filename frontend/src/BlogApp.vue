<template>
  <v-app>
    <v-app-bar color='indigo' dark :density="getDensity()">
      <v-progress-linear
          v-if="showProgress"
          indeterminate
          color="white"
          absolute
      ></v-progress-linear>

      <v-breadcrumbs
          v-if="showSearchButton"
          :items="getBreadcrumbs()"
      >
      </v-breadcrumbs>

      <v-spacer></v-spacer>

      <template v-if="blogStore.isShowSearch">
        <CollapsedSearch :provider="{
              getModelValue: this.getModelValue,
              setModelValue: this.setModelValue,
              getShowSearchButton: this.getShowSearchButton,
              setShowSearchButton: this.setShowSearchButton,
              searchName: this.searchName,
              textFieldVariant: 'solo',
          }"/>
      </template>

    </v-app-bar>

    <v-main>
      <v-container fluid class="ma-0 pa-0" style="height: 100%; width: 100%">
        <router-view />
      </v-container>

    </v-main>

  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import {blog, blog_post, blog_post_name} from "@/router/blogRoutes";
import {hasLength} from "@/utils";
import {
    SEARCH_MODE_POSTS,
    searchStringFacade
} from "@/mixins/searchString";
import {root} from "@/router/routes";
import {mapStores} from "pinia";
import {useBlogStore} from "@/store/blogStore";
import CollapsedSearch from "@/CollapsedSearch.vue";

export default {
  components: {
      CollapsedSearch
  },
  mixins: [
      searchStringFacade(),
  ],
  data: () => ({
    showSearchButton: true,
  }),
  methods: {
    getStore() {
        return this.blogStore
    },
    getBreadcrumbs() {
      const ret = [
        {
          title: 'Videochat',
          disabled: false,
          href: root,
        },
        {
          title: 'Blog',
          disabled: false,
          exactPath: true,
          to: blog,
        },
      ];
      if (this.$route.name == blog_post_name) {
        ret.push(
          {
            title: 'Post #' + this.$route.params.id,
            disabled: false,
            to: blog_post + "/" + this.$route.params.id,
          },
        )
      }
      return ret
    },
    getDensity() {
          return this.isMobile() ? "comfortable" : "compact";
    },
    searchName() {
      if (this.blogStore.searchType == SEARCH_MODE_POSTS) {
          return this.$vuetify.locale.t('$vuetify.search_by_posts')
      }
    },
    getModelValue() {
      return this.searchStringFacade
    },
    setModelValue(v) {
      this.searchStringFacade = v
    },
    getShowSearchButton() {
      return this.showSearchButton
    },
    setShowSearchButton(v) {
      this.showSearchButton = v
    },
  },
  computed: {
    // https://pinia.vuejs.org/cookbook/options-api.html#usage-without-setup
    ...mapStores(useBlogStore),
    hasSearchString() {
      return hasLength(this.searchStringFacade)
    },
    showProgress() {
        return this.blogStore.progressCount > 0
    },
  },
}
</script>

<style lang="stylus">
@import "constants.styl"

.colored-link {
    color: $linkColor;
    text-decoration none
}

.v-breadcrumbs {
  li > a {
    color white
  }
}

.with-pointer {
  cursor pointer
}
</style>
