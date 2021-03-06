<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid v-bind:style="{height: splitpanesHeight + 'px'}">
        <splitpanes ref="spl" class="default-theme" horizontal style="height: 100%"
                    :dbl-click-splitter="false"
                    @pane-add="onPanelAdd(isScrolledToBottom())" @pane-remove="onPanelRemove()" @resize="onPanelResized">
            <pane v-if="isAllowedVideo()" id="videoBlock" min-size="20" v-bind:size="videoSize">
                <ChatVideo :chatDto="chatDto"/>
            </pane>
            <pane v-bind:size="messagesSize">
                <div id="messagesScroller" style="overflow-y: auto; height: 100%" @scroll.passive="onScroll">
                    <v-list  v-if="currentUser">
                        <template v-for="(item, index) in items">
                            <MessageItem :key="item.id" :item="item" :chatId="chatId" :highlight="item.owner.id === currentUser.id"></MessageItem>
                            <v-divider :dark="item.owner.id === currentUser.id"></v-divider>
                        </template>
                    </v-list>
                    <infinite-loading :key="infinityKey" @infinite="infiniteHandler" :identifier="infiniteId" :direction="aDirection" force-use-infinite-wrapper="#messagesScroller" :distance="aDistance">
                        <template slot="no-more"><span/></template>
                        <template slot="no-results">No more messages</template>
                    </infinite-loading>
                </div>
            </pane>
            <pane max-size="70" min-size="12" v-bind:size="editSize">
                <MessageEdit :chatId="chatId" :canBroadcast="canBroadcast"/>
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
        CHAT_DELETED,
        CHAT_EDITED,
        MESSAGE_ADD,
        MESSAGE_DELETED,
        MESSAGE_EDITED,
        USER_TYPING,
        USER_PROFILE_CHANGED,
        LOGGED_IN,
        LOGGED_OUT,
        VIDEO_CALL_CHANGED,
        VIDEO_CALL_KICKED,
        MESSAGE_BROADCAST,
        REFRESH_ON_WEBSOCKET_RESTORED,
    } from "./bus";
    import MessageEdit from "./MessageEdit";
    import {chat_list_name, chat_name, videochat_name} from "./routes";
    import ChatVideo from "./ChatVideo";
    import {getData, getProperData} from "./centrifugeConnection";
    import {mapGetters} from "vuex";
    import {
        GET_USER, SET_CHAT_ID, SET_CHAT_USERS_COUNT,
        SET_SHOW_CALL_BUTTON, SET_SHOW_CHAT_EDIT_BUTTON,
        SET_SHOW_HANG_BUTTON, SET_SHOW_SEARCH, SET_TITLE,
        SET_VIDEO_CHAT_USERS_COUNT
    } from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import {getCorrectUserAvatar} from "./utils";
    import MessageItem from "./MessageItem";
    // import 'splitpanes/dist/splitpanes.css';
    import debounce from "lodash/debounce";
    import throttle from "lodash/throttle";


    const default2 = [80, 20];
    const default3 = [30, 50, 20];

    const directionTop = 'top';
    const directionBottom = 'bottom';

    const maxItemsLength = 100;
    const reduceToLength = 60;

    const calcSplitpanesHeight = () => {
        const appBarHeight = parseInt(document.getElementById("myAppBar").style.height.replace('px', ''));
        const displayableWindowHeight = window.innerHeight;
        const ret = displayableWindowHeight - appBarHeight;
        console.log("splitpanesHeight", ret);
        return ret;
    }

    export default {
        mixins: [infinityListMixin()],
        data() {
            return {
                chatMessagesSubscription: null,
                chatDto: {
                    participantIds:[],
                    participants:[],
                },
                splitpanesHeight: 0,
                aDirection: directionTop,
                infinityKey: 1,
                scrollerDiv: null,
                scrollerProbeCurrent: 0,
                scrollerProbePrevious: 0,
                forbidChangeScrollDirection: false
            }
        },
        computed: {
            aDistance() {
                return this.isTopDirection() ? 0 : 100;
            },
            chatId() {
                return this.$route.params.id
            },
            canBroadcast() {
                return this.chatDto.canBroadcast;
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
            },
        },
        methods: {
            onScroll(e) {
                this.scrollerProbePrevious = this.scrollerProbeCurrent;
                this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
                console.log("onScroll prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

                if (!this.forbidChangeScrollDirection) {
                    Vue.nextTick(() => {
                        this.switchDirection();
                    })
                }
            },
            isTopDirection() {
                return this.aDirection === directionTop
            },
            switchDirection() {
                if (this.scrollerProbeCurrent > this.scrollerProbePrevious && this.isTopDirection()) {
                    this.aDirection = directionBottom;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to bottom");
                } else if (this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
                    this.aDirection = directionTop;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to top");
                }
            },
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
            onPanelAdd(wasScrolled) {
                console.log("On panel add", this.$refs.spl.panes);
                const stored = this.getStored();
                if (stored) {
                    console.log("Restoring from storage", stored);
                    this.$nextTick(() => {
                        if (this.$refs.spl) {
                            this.$refs.spl.panes[0].size = stored[0]; // video
                            this.$refs.spl.panes[1].size = stored[1]; // messages
                            this.$refs.spl.panes[2].size = stored[2]; // edit
                            if (wasScrolled) {
                                this.scrollDown();
                            }
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

            infiniteHandler($state) {
                if (this.items.length) {
                    if (this.isTopDirection()) {
                        this.startingFromItemId = Math.min(...this.items.map(it => it.id));
                    } else {
                        this.startingFromItemId = Math.max(...this.items.map(it => it.id));
                    }
                    console.log("this.startingFromItemId set to", this.startingFromItemId);
                }
                axios.get(`/api/chat/${this.chatId}/message`, {
                    params: {
                        startingFromItemId: this.startingFromItemId,
                        size: pageSize,
                        reverse: this.isTopDirection()
                    },
                }).then(({data}) => {
                    const list = data;
                    if (list.length) {
                        if (this.isTopDirection()) {
                            this.items = list.reverse().concat(this.items);
                        } else {
                            this.items = this.items.concat(list);
                        }
                        this.reduceListIfNeed();
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
            reduceListIfNeed() {
                if (this.items.length > maxItemsLength) {
                    this.forbidChangeScrollDirection = true;
                    setTimeout(() => {
                        console.log("Reducing to", maxItemsLength);
                        if (this.isTopDirection()) {
                            this.items = this.items.slice(0, reduceToLength);
                        } else {
                            this.items = this.items.slice(-reduceToLength);
                        }
                        this.forbidChangeScrollDirection = false;
                    }, 1);
                }
            },
            onNewMessage(dto) {
                if (dto.chatId == this.chatId) {
                    const wasScrolled = this.isScrolledToBottom();
                    this.addItem(dto);
                    if (this.currentUser.id == dto.ownerId || wasScrolled) {
                        this.scrollDown();
                    }
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
                Vue.nextTick(() => {
                    console.log("myDiv.scrollTop", this.scrollerDiv.scrollTop, "myDiv.scrollHeight", this.scrollerDiv.scrollHeight);
                    this.scrollerDiv.scrollTop = this.scrollerDiv.scrollHeight;
                });
            },
            isScrolledToBottom() {
                return this.scrollerDiv.scrollHeight - this.scrollerDiv.scrollTop === this.scrollerDiv.clientHeight
            },
            isScrolledToTop() {
                return this.scrollerDiv.scrollTop === 0
            },
            getInfo() {
                return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
                    console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
                    this.$store.commit(SET_TITLE, data.name);
                    this.$store.commit(SET_CHAT_USERS_COUNT, data.participants.length);
                    this.$store.commit(SET_SHOW_SEARCH, false);
                    this.$store.commit(SET_CHAT_ID, this.chatId);
                    this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, data.canEdit);

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
                            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, data.usersCount);
                        });
                });
            },
            goToChatList() {
                this.$router.push(({name: chat_list_name}))
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.chatDto = dto;
                }
            },
            onChatDelete(dto) {
                if (dto.id == this.chatId) {
                    this.$router.push(({name: chat_list_name}))
                }
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
                const channel = "chatMessages" + this.chatId;
                this.chatMessagesSubscription = this.centrifuge.subscribe(channel, (message) => {
                    // actually it's used for tell server about presence of this client.
                    // also will be used as a global notification, so we just log it
                    const data = getData(message);
                    console.debug("Got message from channel", channel, data);
                    const properData = getProperData(message)
                    if (data.type === "user_typing") {
                        bus.$emit(USER_TYPING, properData);
                    } else if (data.type === "user_broadcast") {
                        bus.$emit(MESSAGE_BROADCAST, properData);
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
            onWsRestoredRefresh() {
                this.searchStringChanged();
            },
            onVideoCallChanged(dto) {
                if (dto.chatId == this.chatId) {
                    this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, dto.usersCount);
                }
            },
            onResizedListener() {
                this.splitpanesHeight = calcSplitpanesHeight();
            }
        },
        created() {
            this.onResizedListener = debounce(this.onResizedListener, 200, {leading:true, trailing:true});
            this.onScroll = throttle(this.onScroll, 200);
        },
        mounted() {
            this.splitpanesHeight = calcSplitpanesHeight();

            window.addEventListener('resize', this.onResizedListener);

            this.subscribe();

            this.$store.commit(SET_TITLE, `Chat #${this.chatId}`);
            this.$store.commit(SET_CHAT_USERS_COUNT, 0);
            this.$store.commit(SET_SHOW_SEARCH, false);
            this.$store.commit(SET_CHAT_ID, this.chatId);
            this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);

            this.getInfo();

            this.$store.commit(SET_SHOW_CALL_BUTTON, true);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);

            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(LOGGED_IN, this.onLoggedIn);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
            bus.$on(VIDEO_CALL_KICKED, this.onVideoCallKicked);
            bus.$on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);

            this.scrollerDiv = document.getElementById("messagesScroller");
        },
        beforeDestroy() {
            window.removeEventListener('resize', this.onResizedListener);

            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(LOGGED_IN, this.onLoggedIn);
            bus.$off(LOGGED_OUT, this.onLoggedOut);
            bus.$off(VIDEO_CALL_KICKED, this.onVideoCallKicked);
            bus.$off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$off(VIDEO_CALL_CHANGED, this.onVideoCallChanged);

            this.unsubscribe();
        },
        destroyed() {
            this.$store.commit(SET_SHOW_CALL_BUTTON, false);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);
            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);
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

.splitpanes__splitter {background-color: #ccc;position: relative; cursor: crosshair}
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
.splitpanes--vertical > .splitpanes__splitter:before {left: -10px;right: -10px;height: 100%;}
.splitpanes--horizontal > .splitpanes__splitter:before {top: -10px;bottom: -10px;width: 100%;}
</style>
