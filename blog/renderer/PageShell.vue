<template>
    <v-app>
        <v-app-bar color='indigo' dark :density="getDensity()">
            <v-breadcrumbs
                :items="getBreadcrumbs()"
            />
            <v-spacer></v-spacer>

        </v-app-bar>

        <v-main>
            <slot />
        </v-main>
    </v-app>
</template>

<script>
    import {hasLength, isMobileBrowser, SEARCH_MODE_POSTS} from "./utils.js";
    import {blog, root} from "./router/routes.js";
    import {usePageContext} from "./usePageContext.js";
    import bus, {SEARCH_STRING_CHANGED} from "./bus/bus.js";

    export default {
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
                return isMobileBrowser() ? "comfortable" : "compact";
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
                        href: blog,
                    },
                ];
                // if (this.$route.name == blog_post_name) {
                //     ret.push(
                //         {
                //             title: 'Post #' + this.$route.params.id,
                //             disabled: false,
                //             to: blog_post + "/" + this.$route.params.id,
                //         },
                //     )
                // }
                return ret
            },
            getProvider() {
                return {
                    getModelValue: this.getModelValue,
                    setModelValue: this.setModelValue,
                    getShowSearchButton: this.getShowSearchButton,
                    setShowSearchButton: this.setShowSearchButton,
                    searchName: this.searchName,
                    textFieldVariant: 'solo',
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
            searchName() {
                return this.$vuetify.locale.t('$vuetify.search_by_posts')
            },
        },
        watch: {
            searchStringFacade: function(newValue, oldValue) {
                console.debug("Route changed from q", SEARCH_MODE_POSTS, oldValue, "->", newValue);
                bus.emit(SEARCH_STRING_CHANGED + '.' + SEARCH_MODE_POSTS, {oldValue: oldValue, newValue: newValue});
            }
        },
        mounted() {
        },
        created() {
        }
    }

</script>


<style lang="stylus">
@import "./styles/constants.styl"

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

.v-breadcrumbs {
    li > a {
        color white
    }
}

.with-pointer {
    cursor pointer
}
</style>
