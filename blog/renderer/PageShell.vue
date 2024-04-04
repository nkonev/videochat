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
    import {isMobileBrowser} from "./utils.js";
    import {blog, root} from "./router/routes.js";

    export default {
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
