<template>
    <v-container class="ma-0 pa-0 my-list-container" fluid>
        <div class="my-message-scroller">
            <h1 v-if="messageDto.is404">404 Not found</h1>
            <MessageItem v-else
                 :id="messageId"
                 :item="messageDto.messageItem"
                 :chatId="chatId"
                 :isInBlog="true"
                 @onreactionclick="onReactionClick"
                 @click="onClickTrap"
            ></MessageItem>
        </div>
    </v-container>
</template>
<script>
import bus, {
    PLAYER_MODAL,
} from "#root/common/bus";
import MessageItem from "#root/common/components/MessageItem.vue";
import {getMessageLink, checkUpByTreeObj} from "#root/common/utils";
import {usePageContext} from "#root/renderer/usePageContext.js";

export default {
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
    components: {
        MessageItem
    },
    methods: {
        onReactionClick() {
            window.location.href = getMessageLink(this.chatId, this.messageId);
        },
        onClickTrap(e) {
            const foundElements = [
                checkUpByTreeObj(e?.target, 1, (el) => {
                    return el?.tagName?.toLowerCase() == "img" ||
                        Array.from(el?.children).find(ch => ch?.classList?.contains("video-in-message-button"))
                })
            ].filter(r => r.found);
            if (foundElements.length) {
                const found = foundElements[foundElements.length - 1].el;
                switch (found?.tagName?.toLowerCase()) {
                    case "img": {
                        bus.emit(PLAYER_MODAL, {canShowAsImage: true, url: found.src})
                        break;
                    }
                    case "div": { // contains video
                        const video = Array.from(found?.children).find(ch => ch?.tagName?.toLowerCase() == "video");
                        bus.emit(PLAYER_MODAL, {canPlayAsVideo: true, url: video.src, previewUrl: video.poster})
                        break;
                    }
                }
            }
        },
    },
    computed: {
        chatId() {
            return this.pageContext.routeParams.id
        },
        messageId() {
            return this.pageContext.routeParams.messageId
        },
    },
}
</script>

<style scoped lang="stylus">
.my-message-scroller {
    height 100%
    width: 100%
    display flex
    flex-direction column
    overflow-y scroll !important
    background white
}
</style>
