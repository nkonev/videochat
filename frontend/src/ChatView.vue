<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid>
        <splitpanes class="default-theme" horizontal style="height: 100%" @resized="onPanesResized">
            <pane v-if="isAllowedVideo()" id="videoBlock">
                <ChatVideo :chatDto="chatDto"/>
            </pane>
            <pane max-size="90" size="80">
                <div id="messagesScroller" style="overflow-y: auto; height: 100%">
                    <v-list>
                        <template v-for="(item, index) in items">
                        <v-list-item
                                :key="item.id"
                                dense
                                class="pr-0 pl-1"
                        >
                            <v-list-item-avatar v-if="item.owner && item.owner.avatar">
                                <v-img :src="item.owner.avatar"></v-img>
                            </v-list-item-avatar>
                            <v-list-item-content @click="onMessageClick(item)" @mousemove="onMessageMouseMove(item)">
                              <v-list-item-subtitle>{{getSubtitle(item)}}</v-list-item-subtitle>
                              <v-list-item-content class="pre-formatted pa-0">{{item.text}}</v-list-item-content>
                            </v-list-item-content>
                            <v-list-item-action>
                                <v-container class="mb-0 mt-0 pb-0 pt-0 mx-2 px-1">
                                    <v-icon class="mr-2" v-if="item.canEdit" color="error" @click="deleteMessage(item)" dark small>mdi-delete</v-icon>
                                    <v-icon v-if="item.canEdit" color="primary" @click="editMessage(item)" dark small>mdi-lead-pencil</v-icon>
                                </v-container>
                            </v-list-item-action>
                        </v-list-item>
                        <v-divider class="ml-15"></v-divider>
                        </template>
                    </v-list>
                    <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId" direction="top" force-use-infinite-wrapper="#messagesScroller" distance="400">
                        <template slot="no-more"><span/></template>
                        <template slot="no-results"><span/></template>
                    </infinite-loading>
                </div>
            </pane>
            <pane max-size="70" size="20">
                <MessageEdit :chatId="chatId"/>
            </pane>
        </splitpanes>
    </v-container>
</template>

<script>
    import axios from "axios";
    import infinityListMixin, {
        findIndex,
        pageSize, replaceInArray
    } from "./InfinityListMixin";
    import Vue from 'vue'
    import bus, {
      CHANGE_PHONE_BUTTON,
      CHANGE_TITLE, CHAT_DELETED,
      CHAT_EDITED,
      MESSAGE_ADD,
      MESSAGE_DELETED,
      MESSAGE_EDITED,
      SET_EDIT_MESSAGE, USER_TYPING,
      VIDEO_LOCAL_ESTABLISHED,
      VIDEO_CHAT_PANES_RESIZED
    } from "./bus";
    import {phoneFactory, titleFactory} from "./changeTitle";
    import MessageEdit from "./MessageEdit";
    import {root_name, videochat_name} from "./routes";
    import ChatVideo from "./ChatVideo";
    import {getData, getProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import 'splitpanes/dist/splitpanes.css'
    import debounce from "lodash/debounce";

    export default {
        mixins:[infinityListMixin()],
        data() {
            return {
                chatMessagesSubscription: null,
                chatDto: {
                    participantIds:[]
                },
                isLoading: false,
            }
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            pageHeight () {
                return document.body.scrollHeight
            },
            ...mapGetters({currentUser: GET_USER})
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

            onVideoChangesHeight() {
                console.log("Adjusting height after video has been shown");
                this.$forceUpdate();
            },

            infiniteHandler($state) {
                if (this.isLoading) {
                    return
                }
                this.isLoading = true;
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
                        // this.items.unshift(...list.reverse());
                        this.items = list.reverse().concat(this.items);
                        return true;
                    } else {
                        return false
                    }
                }).then(value => {
                    this.isLoading = false;
                    if (value) {
                        $state?.loaded();
                    } else {
                        $state?.complete();
                    }
                })
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
                return axios.get(`/api/chat/${this.chatId}`).then(({ data }) => {
                    console.log("Got info about chat", data);
                    bus.$emit(CHANGE_TITLE, titleFactory(data.name, false, data.canEdit, data.canEdit ? this.chatId: null));
                    this.chatDto = data;
                });
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.getInfo()
                }
            },
            onChatDelete(dto) {
                this.$router.push(({ name: root_name}))
            },
            onMessageClick(dto) {
                axios.put(`/api/chat/${this.chatId}/message/read/${dto.id}`);
            },
            isAllowedVideo() {
                return this.currentUser && this.$router.currentRoute.name == videochat_name && this.chatDto && this.chatDto.participantIds && this.chatDto.participantIds.length
            },

            onMessageMouseMove(item) {
                this.onMessageClick(item);
            },
            onPanesResized(obj) {
                bus.$emit(VIDEO_CHAT_PANES_RESIZED, obj);
            },
        },
        mounted() {
            this.chatMessagesSubscription = this.centrifuge.subscribe("chatMessages"+this.chatId, (message) => {
                // actually it's used for tell server about presence of this client.
                // also will be used as a global notification, so we just log it
                const data = getData(message);
                console.debug("Got global notification", data);
                const properData = getProperData(message)
                if (data.type === "user_typing") {
                    bus.$emit(USER_TYPING, properData);
                }
            });

            bus.$emit(CHANGE_TITLE, titleFactory(`Chat #${this.chatId}`, false, true, null));

            this.getInfo();
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true))
            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
            bus.$on(VIDEO_LOCAL_ESTABLISHED, this.onVideoChangesHeight);
        },
        beforeDestroy() {
            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            bus.$off(VIDEO_LOCAL_ESTABLISHED, this.onVideoChangesHeight);

            this.chatMessagesSubscription.unsubscribe();
        },
        destroyed() {
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(false))
        },
        created() {
            this.onMessageMouseMove = debounce(this.onMessageMouseMove, 1000, {leading:true, trailing:false});
            this.infiniteHandler = debounce(this.infiniteHandler, 1000);
        },
        components: {
            MessageEdit,
            ChatVideo,
            Splitpanes, Pane
        }
    }
</script>

<style scoped lang="stylus">
    .pre-formatted {
      white-space pre-wrap
    }

    #chatViewContainer {
        height: calc(100vh - 100px)
        //position: fixed
        //height: calc(100% - 80px)
        //width: calc(100% - 80px)
    }

    #messagesScroller {
        background  white
    }

    #sendButtonContainer {
        background  white
    }

</style>