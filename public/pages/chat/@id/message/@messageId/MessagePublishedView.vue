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
                 @onFilesClicked="onFilesClicked"
            ></MessageItem>
        </div>
    </v-container>
</template>
<script>
import MessageItem from "#root/common/components/MessageItem.vue";
import {getMessageLink, onClickTrap} from "#root/common/utils";
import {usePageContext} from "#root/renderer/usePageContext.js";
import bus, {OPEN_VIEW_FILES_DIALOG} from "#root/common/bus.js";

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
            onClickTrap(e)
        },
        onFilesClicked() {
          const obj = {
            chatId: this.chatId,
            messageId: this.messageId,
            fileItemUuid : this.pageContext.data.messageDto.messageItem.fileItemUuid
          };
          bus.emit(OPEN_VIEW_FILES_DIALOG, obj);
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
