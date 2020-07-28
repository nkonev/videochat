<template>
    <v-card>
        <v-row dense>
            <ChatVideo v-if="isVideo()"/>
            <v-col cols="12">
                <div id="messagesScroller" :style="scrollerHeight()">
                    <v-card-text>
                        <v-list>
                            <template v-for="(item, index) in items">
                            <v-list-item
                                    :key="item.id"
                                    dense
                            >
                                <v-list-item-avatar v-if="item.owner && item.owner.avatar">
                                    <v-img :src="item.owner.avatar"></v-img>
                                </v-list-item-avatar>
                                <v-list-item-content @click="onMessageClick(item)">
                                    <v-list-item-subtitle>{{getSubtitle(item)}}</v-list-item-subtitle>
                                    {{item.text}}
                                </v-list-item-content>
                                <v-list-item-action>
                                    <v-container class="mb-0 mt-0 pb-0 pt-0">
                                        <v-icon class="mr-4" v-if="item.canEdit" color="error" @click="deleteMessage(item)" dark small>mdi-delete</v-icon>
                                        <v-icon v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small>mdi-lead-pencil</v-icon>
                                    </v-container>
                                </v-list-item-action>
                            </v-list-item>
                            <v-divider></v-divider>
                            </template>
                        </v-list>
                        <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId" direction="top">
                            <template slot="no-more"><span/></template>
                            <template slot="no-results"><span/></template>
                        </infinite-loading>
                    </v-card-text>
                    </div>
            </v-col>
        </v-row>
        <MessageEdit :chatId="chatId"/>
    </v-card>
</template>

<script>
    import axios from "axios";
    import infinityListMixin, {
        findIndex,
        pageSize, replaceInArray
    } from "./InfinityListMixin";
    import Vue from 'vue'
    import bus, {
        CHANGE_TITLE,
        CHAT_EDITED,
        MESSAGE_ADD,
        MESSAGE_DELETED,
        MESSAGE_EDITED,
        SET_EDIT_MESSAGE
    } from "./bus";
    import {titleFactory} from "./changeTitle";
    import MessageEdit from "./MessageEdit";
    import {videochat_name} from "./routes";
    import ChatVideo from "./ChatVideo";
    import {getData} from "./centrifugeConnection";

    export default {
        mixins:[infinityListMixin()],
        data() {
            return {
                chatMessagesSubscription: null,
            }
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            pageHeight () {
                return document.body.scrollHeight
            },
        },
        methods: {
            addItem(dto) {
                console.log("Adding item", dto);
                this.items.push(dto);
                this.$forceUpdate();
            },
            changeItem(dto) {
                console.log("Replacing item", dto);
                replaceInArray(this.items, dto);
                this.$forceUpdate();
            },
            removeItem(dto) {
                console.log("Removing item", dto);
                const idxToRemove = findIndex(this.items, dto);
                this.items.splice(idxToRemove, 1);
                this.$forceUpdate();
            },

            deleteMessage(dto){
                axios.delete(`/api/chat/${this.chatId}/message/${dto.id}`)
            },

            editMessage(dto){
                const editMessageDto = {id: dto.id, text: dto.text};
                bus.$emit(SET_EDIT_MESSAGE, editMessageDto);
            },

            scrollerHeight() {
                const maybeScroller = document.getElementById("messagesScroller");
                const maybeSendButton = document.getElementById("sendButtonContainer");

                if (maybeScroller && maybeSendButton) {
                    const topOfScroller = maybeScroller.getBoundingClientRect().top;
                    const sendButtonContainerHeight = maybeSendButton.getBoundingClientRect().height;
                    const availableHeight = window.innerHeight;
                    const newHeight = availableHeight - topOfScroller - sendButtonContainerHeight - 16;
                    if (newHeight > 0) {
                        return `overflow-y: auto; height: ${newHeight}px`
                    }
                }
                return 'overflow-y: auto; height: 240px'
            },
            infiniteHandler($state) {
                axios.get(`/api/chat/${this.chatId}/message`, {
                    params: {
                        page: this.page,
                        size: pageSize,
                        reverse: true
                    },
                }).then(({ data }) => {
                    const list = data;
                    if (list.length) {
                        this.page += 1;
                        // this.items = [...this.items, ...list];
                        this.items.unshift(...list.reverse());
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
            getSubtitle(item) {
                return `${item.owner.login} at ${item.createDateTime}`
            },

            onNewMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.addItem(dto);
                    this.scrollDown();
                } else {
                    console.log("Skipping", dto)
                }
            },
            onDeleteMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.removeItem(dto);
                } else {
                    console.log("Skipping", dto)
                }
            },
            onEditMessage(dto) {
                if (dto.chatId == this.chatId) {
                    this.changeItem(dto);
                } else {
                    console.log("Skipping", dto)
                }
            },
            scrollDown() {
                Vue.nextTick(()=>{
                    var myDiv = document.getElementById("messagesScroller");
                    console.log("myDiv.scrollTop", myDiv.scrollTop, "myDiv.scrollHeight", myDiv.scrollHeight);
                    myDiv.scrollTop = myDiv.scrollHeight;
                });
            },
            getInfo() {
                axios.get(`/api/chat/${this.chatId}`).then(({ data }) => {
                    console.log("Got info about chat", data);
                    bus.$emit(CHANGE_TITLE, titleFactory(data.name, false, data.canEdit, data.canEdit ? this.chatId: null, true));
                });
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.getInfo();
                }
            },
            onMessageClick(dto) {
                axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`);
            },
            isVideo() {
                return this.$router.currentRoute.name == videochat_name
            },

        },
        mounted() {
            this.chatMessagesSubscription = this.centrifuge.subscribe("chatMessages"+this.chatId, (message) => {
                // actually it's used for tell server about presence of this client.
                // also will be used as a global notification, so we just log it
                const data = getData(message);
                console.log("Got global notification", data)
            });

            bus.$emit(CHANGE_TITLE, titleFactory(`Chat #${this.chatId}`, false, true, true));

            this.getInfo();
            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
        },
        beforeDestroy() {
            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            this.chatMessagesSubscription.unsubscribe();
        },
        components: {
            MessageEdit,
            ChatVideo
        }
    }
</script>