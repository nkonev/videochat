<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid>
        <splitpanes class="default-theme" horizontal style="height: 100%">
            <pane v-if="isAllowedVideo()" id="videoBlock" min-size="20" size="20">
                <ChatVideo :chatDto="chatDto"/>
            </pane>
            <pane size="75">
                <div id="messagesScroller" style="overflow-y: auto; height: 100%">
                    <v-list>
                        <template v-for="(item, index) in items">
                            <MessageItem :key="item.id" :item="item" :chatId="chatId"></MessageItem>
                            <v-divider ></v-divider>
                        </template>
                    </v-list>
                    <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId" direction="top" force-use-infinite-wrapper="#messagesScroller" :distance="0">
                        <template slot="no-more"><span/></template>
                        <template slot="no-results">No messages</template>
                    </infinite-loading>
                </div>
            </pane>
            <pane max-size="70" min-size="25" size="25">
                <v-divider v-if="$vuetify.breakpoint.smAndDown"></v-divider>
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
        USER_TYPING,
        VIDEO_LOCAL_ESTABLISHED,
        USER_PROFILE_CHANGED,
        LOGGED_IN, LOGGED_OUT, VIDEO_CALL_CHANGED
    } from "./bus";
    import {phoneFactory, titleFactory} from "./changeTitle";
    import MessageEdit from "./MessageEdit";
    import {chat_list_name, videochat_name} from "./routes";
    import ChatVideo from "./ChatVideo";
    import {getData, getProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import {getCorrectUserAvatar} from "./utils";
    import MessageItem from "./MessageItem";
    import 'splitpanes/dist/splitpanes.css';

    export default {
        mixins: [infinityListMixin()],
        data() {
            return {
                chatMessagesSubscription: null,
                chatDto: {
                    participantIds:[],
                    participants:[],
                },
            }
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER}),
        },
        methods: {
            isAllowedVideo() {
                return this.currentUser && this.$router.currentRoute.name == videochat_name && this.chatDto && this.chatDto.participantIds && this.chatDto.participantIds.length
            },

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

            onVideoChangesHeight() {
                console.log("Adjusting height after video has been shown");
                this.$forceUpdate();
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
                        // this.items.unshift(...list.reverse());
                        this.items = list.reverse().concat(this.items);
                        return true;
                    } else {
                        return false
                    }
                }).then(value => {
                    if (value) {
                        $state?.loaded();
                    } else {
                        $state?.complete();
                    }
                })
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
                    bus.$emit(CHANGE_TITLE, titleFactory(data.name, false, data.canEdit, data.canEdit ? this.chatId: null, this.chatId));
                    this.chatDto = data;
                }).catch(reason => {
                    if (reason.response.status == 404) {
                        this.goToChatList();
                    }
                }).then(() => {
                    axios.get(`/api/chat/${this.chatId}/video/users`)
                        .then(response => response.data)
                        .then(data => {
                            bus.$emit(VIDEO_CALL_CHANGED, data);
                        });
                });
            },
            goToChatList() {
                this.$router.push(({ name: chat_list_name}))
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.getInfo()
                }
            },
            onChatDelete(dto) {
                this.$router.push(({ name: chat_list_name}))
            },
            onUserProfileChanged(user) {
                const patchedUser = user;
                patchedUser.avatar = getCorrectUserAvatar(user.avatar);
                this.items.forEach(item => {
                    if (item.owner.id == user.id) {
                        item.owner = patchedUser;
                    }
                });
            },
            onLoggedIn() {
                this.getInfo();
                this.subscribe();
                this.reloadItems();
            },
            onLoggedOut() {
                this.unsubscribe();
            },
            subscribe() {
                this.chatMessagesSubscription = this.centrifuge.subscribe("chatMessages"+this.chatId, (message) => {
                    // actually it's used for tell server about presence of this client.
                    // also will be used as a global notification, so we just log it
                    const data = getData(message);
                    console.debug("Got global notification", data);
                    const properData = getProperData(message)
                    if (data.type === "user_typing") {
                        bus.$emit(USER_TYPING, properData);
                    }
                    if (data.type === "video_call_changed") {
                        bus.$emit(VIDEO_CALL_CHANGED, properData);
                    }
                });
            },
            unsubscribe() {
                this.chatMessagesSubscription.unsubscribe();
            },
        },
        mounted() {
            this.subscribe();
            bus.$emit(CHANGE_TITLE, titleFactory(`Chat #${this.chatId}`, false, true, null, this.chatId));

            this.getInfo();
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(true, true))
            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
            bus.$on(VIDEO_LOCAL_ESTABLISHED, this.onVideoChangesHeight);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(LOGGED_IN, this.onLoggedIn);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
        },
        beforeDestroy() {
            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            bus.$off(VIDEO_LOCAL_ESTABLISHED, this.onVideoChangesHeight);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(LOGGED_IN, this.onLoggedIn);
            bus.$off(LOGGED_OUT, this.onLoggedOut);

            this.unsubscribe();
        },
        destroyed() {
            bus.$emit(CHANGE_PHONE_BUTTON, phoneFactory(false))
        },
        components: {
            MessageEdit,
            ChatVideo,
            Splitpanes, Pane,
            MessageItem
        }
    }
</script>

<style scoped lang="stylus">
    $mobileWidth = 800px

    .pre-formatted {
      white-space pre-wrap
    }

    #chatViewContainer {
        height: calc(100vh - 70px)
        position: relative
        //height: calc(100% - 80px)
        //width: calc(100% - 80px)
    }
    //
    @media screen and (max-width: $mobileWidth) {
        #chatViewContainer {
            height: calc(100vh - 116px)
        }
    }


    #messagesScroller {
        overflow-y: scroll !important
        background  white
    }

    #sendButtonContainer {
        background white
        // position absolute
        //height 100%
    }

</style>


