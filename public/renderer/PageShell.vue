<template>
    <v-app>
        <v-app-bar color='indigo' dark :density="getDensity()">
            <v-breadcrumbs
                :items="getBreadcrumbs()"
            />
            <v-spacer/>
            <span v-if="shouldShowTitle()" class="app-title-text">{{chatTitle}}</span>
            <v-spacer/>
            <v-btn variant="tonal" v-if="shouldShowGoToChatButton()" @click.prevent="onGoToChat()" :href="chatMessageHref">Go to message</v-btn>

            <template v-if="isShowSearch()">
                <CollapsedSearch :provider="getProvider()"/>
            </template>

        </v-app-bar>

        <v-main>
            <slot />
        </v-main>
    </v-app>
</template>

<script>
    import {hasLength} from "#root/common/utils";
    import {blog, path_prefix, blog_post, videochat} from "#root/common/router/routes.js";
    import bus, {SEARCH_STRING_CHANGED} from "#root/common/bus.js";
    import {usePageContext} from "./usePageContext.js";
    import CollapsedSearch from "#root/common/components/CollapsedSearch.vue";

    export default {
        components: {CollapsedSearch},
        // https://vuejs.org/api/composition-api-setup.html
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
                }
            },

            getModelValue() {
                return this.searchStringFacade
            },
            setModelValue(v) {
                this.searchStringFacade = v

                bus.emit(SEARCH_STRING_CHANGED, v)
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
        },
        computed: {
            chatId() {
                return this.pageContext.routeParams?.id
            },
            messageId() {
                return this.pageContext.routeParams?.messageId
            },
        },
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

</style>

<style lang="stylus" scoped>
.app-title-text {
    font-size: .875rem;
    font-weight: 500;
    letter-spacing: .09em;
    height: 1.6em;
    white-space: break-spaces;
    overflow: hidden;
}

</style>
