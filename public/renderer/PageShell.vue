<template>
    <v-app>
        <v-app-bar color='indigo' dark :density="getDensity()">
            <v-breadcrumbs
                :items="getBreadcrumbs()"
            />
            <v-spacer></v-spacer>
            <v-btn variant="tonal" v-if="shouldShowGoToChatButton()" @click.prevent="onGoToChat()" :href="chatMessageHref">Go to message</v-btn>

        </v-app-bar>

        <v-main>
            <slot />
        </v-main>
    </v-app>
</template>

<script>
    import {hasLength} from "#root/common/utils";
    import {blog, path_prefix, blog_post, videochat} from "#root/common/router/routes.js";
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
                return "compact";
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
                        href: path_prefix + blog + "/",
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
                return hasLength(this.messageId)
            },
            onGoToChat() {
                window.location.href = this.chatMessageHref
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
