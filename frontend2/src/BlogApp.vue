<template>
  <v-app>
    <v-app-bar color='indigo' dark :density="getDensity()">
      <v-breadcrumbs
        :items="getBreadcrumbs()"
      >
      </v-breadcrumbs>

      <v-spacer></v-spacer>

      <template>
        <v-btn v-if="showSearchButton && isMobile()" icon :title="searchName()" @click="onOpenSearch()">
          <v-icon>{{ hasSearchString ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
        </v-btn>
        <v-card v-if="!showSearchButton || !isMobile()" variant="plain" min-width="330"  style="margin-left: 1.2em; margin-right: 2px">
          <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line v-model="searchStringFacade" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" :label="searchName()"></v-text-field>
        </v-card>
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

export default {
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
    resetInput() {
      this.searchString = null;
      this.showSearchButton = true;
    },
    getBreadcrumbs() {
      const ret = [
        {
          text: 'Videochat',
          disabled: false,
          href: root,
        },
        {
          text: 'Blog',
          disabled: false,
          exactPath: true,
          to: blog,
        },
      ];
      if (this.$route.name == blog_post_name) {
        ret.push(
          {
            text: 'Post',
            disabled: false,
            to: blog_post,
          },
        )
      }
      return ret
    },
    onOpenSearch() {
      this.showSearchButton = false;
    },
    searchName() {
        if (this.blogStore.searchType == SEARCH_MODE_POSTS) {
            return this.$vuetify.locale.t('$vuetify.search_by_posts')
        }
    },
    getDensity() {
          return this.isMobile() ? "comfortable" : "compact";
    },
  },
  computed: {
    // https://pinia.vuejs.org/cookbook/options-api.html#usage-without-setup
    ...mapStores(useBlogStore),
    searchIcon() {
      if (this.blogStore.searchType == SEARCH_MODE_POSTS) {
          return 'mdi-forum'
      }
    },
    hasSearchString() {
      return hasLength(this.searchString)
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
</style>
