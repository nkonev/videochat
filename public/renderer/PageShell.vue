<template>
    <v-app>
        <v-app-bar color='indigo' dark :density="getDensity()">
          <v-container fluid class="ma-0 pa-0 d-flex">
            <template v-if="getShowSearchButton()">
                <v-breadcrumbs
                    :items="getBreadcrumbs()"
                />
                <v-spacer/>
                <div class="app-title ml-2 align-self-center" v-if="shouldShowTitle()">
                  <div class="app-title-text pl-1">{{chatTitle}}</div>
                </div>
                <v-spacer/>
                <span v-if="shouldShowAboutTitle()" class="mr-4 align-content-center app-title-text"><a class="v-breadcrumbs-item--link" :href="aboutPostTitleHref">{{aboutPostTitle}}</a></span>
                <div class="ma-0 pa-0 mr-1 align-content-center">
                  <v-btn variant="tonal" v-if="shouldShowGoToChatButton()" @click.prevent="onGoToChat()" :href="chatMessageHref">Go to message</v-btn>
                </div>
            </template>
            <template v-if="isShowSearch()">
                <CollapsedSearch :provider="getProvider()"/>
            </template>
          </v-container>
        </v-app-bar>

        <v-main>
          <template v-if="pageLoading">
            <v-progress-circular
                class="ma-4"
                color="primary"
                indeterminate
            ></v-progress-circular>
          </template>
          <template v-else>
            <slot />
          </template>

          <PlayerModal/>
          <FileListModal/>
        </v-main>
    </v-app>
</template>

<script>
    import {hasLength} from "#root/common/utils";
    import {blog, path_prefix, blog_post, videochat} from "#root/common/router/routes.js";
    import bus, {SEARCH_STRING_CHANGED, SET_LOADING, SET_SET_SEARCH_STRING_NO_EMIT} from "#root/common/bus.js";
    import {usePageContext} from "./usePageContext.js";
    import CollapsedSearch from "#root/common/components/CollapsedSearch.vue";
    import PlayerModal from "#root/common/components/PlayerModal.vue";
    import FileListModal from "#root/common/components/FileListModal.vue";

    export default {
        components: {
          PlayerModal,
          FileListModal,
          CollapsedSearch,
        },
        // https://vuejs.org/api/composition-api-setup.html
        setup() {
            const pageContext = usePageContext();

            // expose to template and other options API hooks
            return {
                pageContext
            }
        },
        data() {
          return {
            chatMessageHref: this.pageContext.data.chatMessageHref,
            chatTitle: this.pageContext.data.chatTitle,
            aboutPostTitle: this.pageContext.data.header?.aboutPostTitle,
            aboutPostId: this.pageContext.data.header?.aboutPostId,
            showSearchButton: this.pageContext.data.showSearchButton,
            pageLoading: false,
            searchStringFacade: this.pageContext.data.searchStringFacade,
          }
        },
        methods: {
            getDensity() {
                return this.isMobile() ? "comfortable" : "compact";
            },
            isMobile() {
                return this.pageContext.isMobile
            },
            getBreadcrumbs() {
                const ret = [
                    {
                        title: 'Videochat',
                        disabled: false,
                        href: videochat,
                    },
                ];

                if (this.pageContext.urlParsed && this.pageContext.urlParsed.pathname.startsWith(blog)) {

                    ret.push({
                        title: 'Blog',
                        disabled: false,
                        exactPath: true,
                        href: path_prefix + blog,
                    })

                    if (hasLength(this.chatId)) {
                        ret.push(
                            {
                                title: 'Post #' + this.chatId,
                                disabled: false,
                                href: path_prefix + blog_post + "/" + this.chatId,
                            },
                        )
                    }
                }
                return ret
            },
            shouldShowGoToChatButton() {
                return hasLength(this.$data.chatMessageHref)
            },
            onGoToChat() {
                window.location.href = this.$data.chatMessageHref
            },
            isShowSearch() {
                return !hasLength(this.chatId) && !hasLength(this.messageId)
            },

            getProvider() {
                return {
                    getModelValue: this.getModelValue,
                    setModelValue: this.setModelValue,
                    getShowSearchButton: this.getShowSearchButton,
                    setShowSearchButton: this.setShowSearchButton,
                    searchName: this.searchName,
                    searchIcon: this.searchIcon,
                    textFieldVariant: 'solo',
                    switchSearchType: false,
                }
            },

            getModelValue() {
                return this.searchStringFacade
            },
            setModelValue(v) {
                this.setModelValueNoEmit(v);

                bus.emit(SEARCH_STRING_CHANGED, v);
            },
            setModelValueNoEmit(v) {
              // comes from public/pages/blog/+data.js
              this.searchStringFacade = v
            },
            getShowSearchButton() {
                return this.$data.showSearchButton
            },
            setShowSearchButton(v) {
                this.$data.showSearchButton = v
            },
            searchName() {
                return "Search by blogs"
            },
            searchIcon() {
                return "mdi-postage-stamp"
            },

            shouldShowTitle() {
                return hasLength(this.$data.chatTitle)
            },
            shouldShowAboutTitle() {
                return hasLength(this.$data.aboutPostTitle)
            },
            setLoading(v) {
              this.$data.pageLoading = v
            },
        },
        computed: {
            chatId() {
                return this.pageContext.routeParams?.id
            },
            messageId() {
                return this.pageContext.routeParams?.messageId
            },
            aboutPostTitleHref() {
                return path_prefix + blog_post + "/" + this.aboutPostId
            },
        },
        mounted() {
          bus.on(SET_LOADING, this.setLoading)
          bus.on(SET_SET_SEARCH_STRING_NO_EMIT, this.setModelValueNoEmit)
        },
        beforeUnmount() {
          bus.off(SET_LOADING, this.setLoading)
          bus.off(SET_SET_SEARCH_STRING_NO_EMIT, this.setModelValueNoEmit)
        }
    }

</script>


<style lang="stylus">
@import "../common/styles/constants.styl"

// removes extraneous scroll at right side of the screen on Chrome
html {
    overflow-y: unset !important;
}

.with-space {
    white-space: pre;
}

.colored-link {
    color: $linkColor;
    text-decoration none
}

.gray-link {
    color: $grayColor;
    text-decoration none
}

.nodecorated-link {
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

.my-list-container {
    height: calc(100dvh - 48px)
    top: 48px
    position: absolute
}

.caption-small {
  color:rgba(0, 0, 0, .6);
  font-size: 0.9rem;
  font-weight: 500;
  line-height: 1rem;
  display: inherit
}

</style>

<style lang="stylus" scoped>

.app-title {
  width: 100%;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}

.app-title-text {
    display: inline;
    font-size: .875rem;
    font-weight: 500;
    letter-spacing: .09em;
    line-height: 1.75em;
}

</style>
