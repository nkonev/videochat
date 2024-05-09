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
    import {hasLength, isMobileBrowser, SEARCH_MODE_POSTS} from "../common/utils.js";
    import {blog, root, blog_post, blog_post_name} from "../common/router/routes.js";
    import {usePageContext} from "./usePageContext.js";

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
                if (this.pageContext.urlOriginal.startsWith(blog_post)) {
                    const id = this.pageContext.urlOriginal.replace(blog_post+"/", "");
                    ret.push(
                        {
                            title: 'Post #' + id,
                            disabled: false,
                            href: blog_post + "/" + id,
                        },
                    )
                }
                return ret
            },
        },
        mounted() {
        },
        created() {
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

.v-breadcrumbs {
    li > a {
        color white
    }
}

.with-pointer {
    cursor pointer
}
</style>
