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
    import { navigate } from 'vike/client/router';
    import {usePageContext} from "./usePageContext.js";
    import CollapsedSearch from "./CollapsedSearch.vue";

    export default {
        // https://vuejs.org/api/composition-api-setup.html
        setup() {
            const pageContext = usePageContext();

            // expose to template and other options API hooks
            return {
                pageContext
            }
        },
        components: {CollapsedSearch},
        data() {
            return this.pageContext.data;
        },
        computed: {
            searchStringFacade: {
                get() {
                    if (typeof window === 'undefined') {
                        return this.pageContext.urlParsed.search[SEARCH_MODE_POSTS];
                    } else {
                        // TODO fix mismatch

                        // idea from https://github.com/vikejs/vike/issues/1231
                        // see also https://stackoverflow.com/questions/4570093/how-to-get-notified-about-changes-of-the-history-via-history-pushstate/4585031#4585031
                        const url = new URL(location);
                        const ret = url.searchParams.get(SEARCH_MODE_POSTS);
                        // console.log("ret is", url);
                        return ret;
                    }
                },
                set(newVal) {
                    // if (hasLength(newVal)) {
                    //     navigate(blog + '?' + SEARCH_MODE_POSTS + "=" + newVal)
                    // } else {
                    //     navigate(blog)
                    // }
                    const url = new URL(location);
                    url.searchParams.set(SEARCH_MODE_POSTS, newVal);
                    history.pushState({}, "", url);
                }
            }
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
