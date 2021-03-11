<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid>
        <splitpanes ref="spl" class="default-theme" horizontal style="height: 100%"
                    @pane-add="onPanelAdd" @pane-remove="onPanelRemove" @resize="onPanelResized">
            <pane v-if="isAllowedVideo()" id="videoBlock" min-size="20" v-bind:size="videoSize">
                <ChatVideo :chatDto="chatDto"/>
            </pane>
            <pane v-bind:size="messagesSize">
                <div id="messagesScroller" style="overflow-y: auto; height: 100%">
                    <v-list>
                        <template v-for="(item, index) in items">
                            <v-divider></v-divider>
                            <MessageItem :key="item.id" :item="item" :chatId="chatId"></MessageItem>
                        </template>
                    </v-list>
                    <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId" direction="top" force-use-infinite-wrapper="#messagesScroller" :distance="0">
                        <template slot="no-more"><span/></template>
                        <template slot="no-results">No more messages</template>
                    </infinite-loading>
                </div>
            </pane>
            <pane max-size="70" min-size="20" v-bind:size="editSize">
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
        LOGGED_IN, LOGGED_OUT, VIDEO_CALL_CHANGED, VIDEO_CALL_KICKED
    } from "./bus";
    import {phoneFactory, titleFactory} from "./changeTitle";
    import MessageEdit from "./MessageEdit";
    import {chat_list_name, chat_name, videochat_name} from "./routes";
    import ChatVideo from "./ChatVideo";
    import {getData, getProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import {getCorrectUserAvatar} from "./utils";
    import MessageItem from "./MessageItem";
    // import 'splitpanes/dist/splitpanes.css';

    const default2 = [80, 20];
    const default3 = [30, 50, 20];

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
            videoSize() {
                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? default3 : default2)
                    stored = this.getStored();
                }
                if (this.isAllowedVideo()) {
                    return stored[0]
                } else {
                    console.error("Unable to get video size if video is not enabled");
                    return 0
                }
            },
            messagesSize() {
                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? default3 : default2)
                    stored = this.getStored();
                }
                if (this.isAllowedVideo()) {
                    return stored[1]
                } else {
                    return stored[0]
                }
            },
            editSize() {
                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? default3 : default2)
                    stored = this.getStored();
                }
                if (this.isAllowedVideo()) {
                    return stored[2]
                } else {
                    return stored[1]
                }
            }
        },
        methods: {
            getStored() {
                const mbItem = this.isAllowedVideo() ? localStorage.getItem('3panels') : localStorage.getItem('2panels');
                if (!mbItem) {
                    return null;
                } else {
                    return JSON.parse(mbItem);
                }
            },
            saveToStored(arr) {
                if (this.isAllowedVideo()) {
                    localStorage.setItem('3panels', JSON.stringify(arr));
                } else {
                    localStorage.setItem('2panels', JSON.stringify(arr));
                }
            },
            onPanelAdd() {
                console.log("On panel add", this.$refs.spl.panes);
                const stored = this.getStored();
                if (stored) {
                    console.log("Restoring from storage", stored);
                    this.$nextTick(() => {
                        if (this.$refs.spl) {
                            this.$refs.spl.panes[0].size = stored[0]; // video
                            this.$refs.spl.panes[1].size = stored[1]; // messages
                            this.$refs.spl.panes[2].size = stored[2]; // edit
                        }
                    })
                } else {
                    console.error("Store is null");
                }
            },
            onPanelRemove() {
                console.log("On panel removed", this.$refs.spl.panes);
                const stored = this.getStored();
                if (stored) {
                    console.log("Restoring from storage", stored);
                    this.$nextTick(() => {
                        if (this.$refs.spl) {
                            this.$refs.spl.panes[0].size = stored[0]; // messages
                            this.$refs.spl.panes[1].size = stored[1]; // edit
                        }
                    })
                } else {
                    console.error("Store is null");
                }
            },
            onPanelResized() {
                // console.log("On panel resized", this.$refs.spl.panes);
                this.saveToStored(this.$refs.spl.panes.map(i => i.size));
            },
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
                    bus.$emit(CHANGE_TITLE, titleFactory(data.name, false, data.canEdit, data.canEdit ? this.chatId: null, this.chatId, data.participants.length));
                    this.chatDto = data;
                }).catch(reason => {
                    if (reason.response.status == 404) {
                        this.goToChatList();
                    }
                }).then(() => {
                    axios.get(`/api/video/${this.chatId}/users`)
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
            onVideoCallKicked(e) {
                if (this.$route.name == videochat_name && e.chatId == this.chatId) {
                    console.log("kicked");
                    this.$router.push({name: chat_name});
                }
            },
        },
        mounted() {
            this.subscribe();
            bus.$emit(CHANGE_TITLE, titleFactory(`Chat #${this.chatId}`, false, true, null, this.chatId, null));

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
            bus.$on(VIDEO_CALL_KICKED, this.onVideoCallKicked);
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
            bus.$off(VIDEO_CALL_KICKED, this.onVideoCallKicked);

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
        height: calc(100vh - 68px)
        position: relative
        //height: calc(100% - 80px)
        //width: calc(100% - 80px)
    }
    //
    //@media screen and (max-width: $mobileWidth) {
    //    #chatViewContainer {
    //        height: calc(100vh - 116px)
    //    }
    //}


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

<style lang="stylus">
.splitpanes {background-color: #f8f8f8;}

.splitpanes__splitter {background-color: #ccc;position: relative;}
.splitpanes__splitter:before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    transition: opacity 0.4s;
    background-color: rgba(255, 0, 0, 0.3);
    opacity: 0;
    z-index: 1;
}
.splitpanes__splitter:hover:before {opacity: 1;}
.splitpanes--vertical > .splitpanes__splitter:before {left: -20px;right: -20px;height: 100%;}
.splitpanes--horizontal > .splitpanes__splitter:before {top: -20px;bottom: -20px;width: 100%;}
</style>
