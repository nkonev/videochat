<template>
    <v-app>
        <v-app-bar color='indigo' dark :density="getDensity()">
            <v-breadcrumbs
                :items="getBreadcrumbs()"
            />
            <v-spacer></v-spacer>

            <template v-if="true">
                <CollapsedSearch :provider="getProvider()"/>
            </template>
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
    import CollapsedSearch from "./CollapsedSearch.vue";
    import { computed, ref } from "vue";

    export default {
        // https://vuejs.org/api/composition-api-setup.html
        setup() {
            const pageContext = usePageContext();

            const searchString = ref("");

            searchString.value = typeof window === 'undefined' ? pageContext.urlParsed.search[SEARCH_MODE_POSTS] : new URL(location).searchParams.get(SEARCH_MODE_POSTS);

            const searchStringFacade = computed({
                get: () => {
                    return searchString.value
                },
                set: val => {
                    searchString.value = val

                    const url = new URL(location);
                    if (hasLength(val)) {
                        url.searchParams.set(SEARCH_MODE_POSTS, val);
                    } else {
                        url.searchParams.delete(SEARCH_MODE_POSTS);
                    }
                    history.pushState({}, "", url);
                }
            })

            // expose to template and other options API hooks
            return {
                pageContext,
                searchStringFacade,
            }
        },
        components: {CollapsedSearch},
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
