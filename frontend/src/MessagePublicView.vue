<template>
    <v-container v-if="loaded" class="ma-0 pa-0" :style="heightWithoutAppBar" fluid>
        <div class="my-message-scroller">
            <h1 v-if="is404">404 Not found</h1>
            <MessageItem v-else
                 :id="messageId"
                 :item="messageItemDto"
                 :chatId="chatId"
                 :isInBlog="true"
            ></MessageItem>
        </div>
    </v-container>
</template>
<script>
import MessageItem from "@/MessageItem.vue";
import axios from "axios";
import {setTitle} from "@/utils.js";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore.js";
import heightMixin from "@/mixins/heightMixin.js";

export default {
    mixins: [
        heightMixin(),
    ],
    data() {
        return {
            loaded: false,
            messageItemDto: { },
            is404: false,
        }
    },
    components: {
        MessageItem
    },
    methods: {
        loadData() {
            this.chatStore.incrementProgressCount();
            return axios.get(`/api/chat/public/${this.chatId}/message/${this.messageId}`).then((response) => {
                if (response.status == 204) {
                    this.is404 = true;
                    setTitle("Page not found");
                } else {
                    this.messageItemDto = response.data.message;
                    this.chatStore.title = response.data.title;
                    setTitle(response.data.title);
                }
            }).finally(()=>{
                this.loaded = true
                this.chatStore.decrementProgressCount();
            });

        }
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        messageId() {
            return this.$route.params.messageId
        }
    },
    mounted() {
        this.loadData();
    }
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
