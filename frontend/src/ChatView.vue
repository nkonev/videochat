<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid>
        <splitpanes ref="spl" :class="['default-theme', this.isAllowedVideo() ? 'panes3' : 'panes2']" horizontal style="height: 100%"
                    :dbl-click-splitter="false"
                    @pane-add="onPanelAdd(isScrolledToBottom())" @pane-remove="onPanelRemove()" @resize="onPanelResized(isScrolledToBottom())">
            <pane v-if="isAllowedVideo()" id="videoBlock" min-size="20" v-bind:size="videoSize">
                <ChatVideo :chatDto="chatDto"/>
            </pane>
            <pane v-bind:size="messagesSize">
                <div id="messagesScroller" style="overflow-y: auto; height: 100%" @scroll.passive="onScroll">
                    <v-list  v-if="currentUser">
                        <template v-for="(item, index) in items">
                            <MessageItem :key="item.id" :item="item" :chatId="chatId" :my="item.owner.id === currentUser.id" :highlight="item.id == highlightMessageId"></MessageItem>
                        </template>
                    </v-list>
                    <infinite-loading :key="infinityKey" @infinite="infiniteHandler" :identifier="infiniteId" :direction="aDirection" force-use-infinite-wrapper="#messagesScroller" :distance="aDistance">
                        <template slot="no-more"><span/></template>
                        <template slot="no-results"><span/></template>
                    </infinite-loading>
                </div>

                <v-btn
                    v-if="!isMobile() && !isScrolledToBottom()"
                    color="primary"
                    fab
                    dark
                    style="position: relative; bottom: 66px; margin-left: 100%; right: 66px; z-index: 2"
                    @click="onClickScrollDown()"
                >
                    <v-icon>mdi-chevron-down</v-icon>
                </v-btn>
            </pane>
            <pane max-size="70" min-size="12" v-bind:size="editSize" v-if="!isMobile()">
                <MessageEdit :chatId="chatId"/>
            </pane>
        </splitpanes>
        <v-btn v-if="isMobile()"
            color="primary"
            fab
            dark
            bottom
            right
            fixed
            @click="openNewMessageDialog()"
        >
            <v-icon>mdi-message-plus</v-icon>
        </v-btn>

        <v-tooltip v-if="writingUsers.length || broadcastMessage" :activator="'#chatViewContainer'" bottom v-model="showTooltip" :key="tooltipKey">
            <span v-if="!broadcastMessage">{{writingUsers.map(v=>v.login).join(', ')}} {{ $vuetify.lang.t('$vuetify.user_is_writing') }}</span>
            <span v-else>{{broadcastMessage}}</span>
        </v-tooltip>
    </v-container>
</template>

<script>
    import axios from "axios";
    import InfiniteLoading from 'vue-infinite-loading';
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
        VIDEO_CALL_USER_COUNT_CHANGED,
        MESSAGE_BROADCAST,
        REFRESH_ON_WEBSOCKET_RESTORED,
        OPEN_EDIT_MESSAGE,
        CHAT_ADD,
        PROFILE_SET,
        FILE_UPLOADED,
        PARTICIPANT_ADDED,
        PARTICIPANT_DELETED, PARTICIPANT_EDITED,
    } from "./bus";
    import {chat_list_name, chat_name, videochat_name} from "./routes";
    import MessageEdit from "./MessageEdit";
    import ChatVideo from "./ChatVideo";

    import {mapGetters} from "vuex";

    import {
        GET_USER,
        SET_CAN_BROADCAST_TEXT_MESSAGE,
        SET_CAN_MAKE_RECORD,
        SET_CHAT_ID,
        SET_CHAT_USERS_COUNT,
        SET_SEARCH_NAME,
        SET_SHOW_CALL_BUTTON,
        SET_SHOW_CHAT_EDIT_BUTTON,
        SET_SHOW_HANG_BUTTON,
        SET_SHOW_RECORD_START_BUTTON,
        SET_SHOW_RECORD_STOP_BUTTON,
        SET_SHOW_SEARCH,
        SET_TITLE,
        SET_VIDEO_CHAT_USERS_COUNT
    } from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import {findIndex, findIndexNonStrictly, hasLength, replaceInArray} from "./utils";
    import MessageItem from "./MessageItem";
    // import 'splitpanes/dist/splitpanes.css';
    import debounce from "lodash/debounce";
    import throttle from "lodash/throttle";
    import queryMixin from "@/queryMixin";
    import graphqlSubscriptionMixin from "./graphqlSubscriptionMixin"


    const defaultDesktopWithoutVideo = [80, 20];
    const defaultDesktopWithVideo = [30, 50, 20];

    const defaultMobileWithoutVideo = [100];
    const defaultMobileWithVideo = [40, 60];

    const KEY_DESKTOP_WITH_VIDEO_PANELS = 'desktopWithVideo';
    const KEY_DESKTOP_WITHOUT_VIDEO_PANELS = 'desktopWithoutVideo'
    const KEY_MOBILE_WITH_VIDEO_PANELS = 'mobileWithVideo';
    const KEY_MOBILE_WITHOUT_VIDEO_PANELS = 'mobileWithoutVideo'

    const directionTop = 'top';
    const directionBottom = 'bottom';

    const maxItemsLength = 200;
    const reduceToLength = 100;

    const pageSize = 40;

    const scrollingThreshold = 200; // px

    let writingUsersTimerId;

    const getChatEventsData = (message) => {
        return message.data?.chatEvents
    };

    export default {
        mixins: [
            queryMixin(),
            graphqlSubscriptionMixin('chatEvents')
        ],
        data() {
            return {
                startingFromItemId: null,
                items: [],
                infiniteId: +new Date(),
                highlightMessageId: null,

                chatDto: {
                    participantIds:[],
                    participants:[],
                },
                aDirection: directionTop,
                infinityKey: 1,
                scrollerDiv: null,
                scrollerProbeCurrent: 0,
                scrollerProbePrevious: 0,
                scrollerProbePreviousPrevious: 0,

                writingUsers: [],
                showTooltip: true,
                broadcastMessage: null,
                tooltipKey: 0,
                hash: null,
            }
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER}),
            videoSize() {
                let defaultWithVideo;
                let defaultWithoutVideo;
                if (!this.isMobile()) {
                    defaultWithVideo = defaultDesktopWithVideo;
                    defaultWithoutVideo = defaultDesktopWithoutVideo;
                } else {
                    defaultWithVideo = defaultMobileWithVideo;
                    defaultWithoutVideo = defaultMobileWithoutVideo;
                }

                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? defaultWithVideo : defaultWithoutVideo)
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
                let defaultWithVideo;
                let defaultWithoutVideo;
                if (!this.isMobile()) {
                    defaultWithVideo = defaultDesktopWithVideo;
                    defaultWithoutVideo = defaultDesktopWithoutVideo;
                } else {
                    defaultWithVideo = defaultMobileWithVideo;
                    defaultWithoutVideo = defaultMobileWithoutVideo;
                }

                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? defaultWithVideo : defaultWithoutVideo)
                    stored = this.getStored();
                }
                if (this.isAllowedVideo()) {
                    return stored[1]
                } else {
                    return stored[0]
                }
            },
            editSize() {
                // not need here because it's not used in mobile

                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(this.isAllowedVideo() ? defaultDesktopWithVideo : defaultDesktopWithoutVideo)
                    stored = this.getStored();
                }
                if (this.isAllowedVideo()) {
                    return stored[2]
                } else {
                    return stored[1]
                }
            },
            aDistance() {
                return this.isTopDirection() ? 0 : 100
            },
            userIsSet() {
                return !!this.currentUser
            },
            hasHash() {
                return hasLength(this.hash)
            }
        },
        methods: {
            // not working until you will change this.items list
            reloadItems() {
              this.infiniteId += 1;
              console.log("Resetting infinite loader", this.infiniteId);
            },
            searchStringChanged(searchString) {
                this.resetVariables();
                this.reloadItems();
            },

            onScroll(e) {
                this.scrollerProbePreviousPrevious = this.scrollerProbePrevious;
                this.scrollerProbePrevious = this.scrollerProbeCurrent;
                this.scrollerProbeCurrent = this.scrollerDiv.scrollTop;
                console.debug("onScroll prevPrev=", this.scrollerProbePreviousPrevious , " prev=", this.scrollerProbePrevious, "cur=", this.scrollerProbeCurrent);

                this.trySwitchDirection();
            },
            isTopDirection() {
                return this.aDirection === directionTop
            },
            trySwitchDirection() {
                if (this.scrollerProbeCurrent > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbePreviousPrevious && this.isTopDirection()) {
                    this.aDirection = directionBottom;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to bottom");
                } else if (this.scrollerProbePreviousPrevious > this.scrollerProbePrevious && this.scrollerProbePrevious > this.scrollerProbeCurrent && !this.isTopDirection()) {
                    this.aDirection = directionTop;
                    this.infinityKey++;
                    console.log("Infinity scrolling direction has been changed to top");
                } else {
                    console.log("Infinity scrolling direction has been remained untouched");
                }
            },
            getStored() {
                let keyWithVideo;
                let keyWithoutVideo;
                if (!this.isMobile()) {
                    keyWithVideo = KEY_DESKTOP_WITH_VIDEO_PANELS;
                    keyWithoutVideo = KEY_DESKTOP_WITHOUT_VIDEO_PANELS;
                } else {
                    keyWithVideo = KEY_MOBILE_WITH_VIDEO_PANELS;
                    keyWithoutVideo = KEY_MOBILE_WITHOUT_VIDEO_PANELS;
                }

                const mbItem = this.isAllowedVideo() ? localStorage.getItem(keyWithVideo) : localStorage.getItem(keyWithoutVideo);
                if (!mbItem) {
                    return null;
                } else {
                    return JSON.parse(mbItem);
                }
            },
            saveToStored(arr) {
                let keyWithVideo;
                let keyWithoutVideo;
                if (!this.isMobile()) {
                    keyWithVideo = KEY_DESKTOP_WITH_VIDEO_PANELS;
                    keyWithoutVideo = KEY_DESKTOP_WITHOUT_VIDEO_PANELS;
                } else {
                    keyWithVideo = KEY_MOBILE_WITH_VIDEO_PANELS;
                    keyWithoutVideo = KEY_MOBILE_WITHOUT_VIDEO_PANELS;
                }

                if (this.isAllowedVideo()) {
                    localStorage.setItem(keyWithVideo, JSON.stringify(arr));
                } else {
                    localStorage.setItem(keyWithoutVideo, JSON.stringify(arr));
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
                            if (this.$refs.spl.panes[2]) {
                                this.$refs.spl.panes[2].size = stored[2]; // edit
                            }
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
                            if (this.$refs.spl.panes[1]) {
                                this.$refs.spl.panes[1].size = stored[1]; // edit
                            }
                        }

                    })
                } else {
                    console.error("Store is null");
                }
            },
            onPanelResized(wasScrolled) {
                // console.log("On panel resized", this.$refs.spl.panes);
                this.saveToStored(this.$refs.spl.panes.map(i => i.size));
                this.$nextTick(()=>{
                    if (wasScrolled) {
                        this.scrollDown();
                    }
                })
            },
            isAllowedVideo() {
                return this.currentUser && this.$router.currentRoute.name == videochat_name && this.chatDto && this.chatDto.participantIds && this.chatDto.participantIds.length
            },

            addItem(dto) {
                console.log("Adding item", dto);
                this.items.push(dto);
                this.reduceListIfNeed();
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

                if (!this.userIsSet) {
                    $state.complete();
                    return
                }

                axios.get(`/api/chat/${this.chatId}/message`, {
                    params: {
                        startingFromItemId: this.hasHash ? this.highlightMessageId : this.startingFromItemId,
                        size: pageSize,
                        reverse: this.isTopDirection(),
                        searchString: this.searchString,
                        hasHash: this.hasHash
                    },
                }).then(({data}) => {
                    const list = data;
                    if (list.length) {
                        if (this.isTopDirection()) {
                            this.items = list.reverse().concat(this.items);
                        } else {
                            this.items = this.items.concat(list);
                        }
                        if (this.items.length > pageSize) {
                            this.clearRouteHash();
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
                    if (this.hasHash) {
                        try {
                            this.$vuetify.goTo('#' + this.hash, {container: this.scrollerDiv, duration: 0});
                        } catch (err) {
                            console.debug("Didn't scrolled", err)
                        }
                    }
                    this.hash = null;
                })
            },
            reduceListIfNeed() {
                if (this.items.length > maxItemsLength) {
                    setTimeout(() => {
                        console.log("Reducing to", maxItemsLength);
                        if (this.isTopDirection()) {
                            this.items = this.items.slice(0, reduceToLength);
                        } else {
                            this.items = this.items.slice(-reduceToLength);
                        }
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
                    const isScrolled = this.isScrolledToBottom();
                    this.changeItem(dto);
                    if (isScrolled) {
                        this.scrollDown();
                    }
                } else {
                    console.log("Skipping", dto)
                }
            },
            onClickScrollDown() {
                // condition is a dummy heuristic (because right now doe to outdated vue-infinite-loading we cannot scroll down several times. nevertheless I think it's a pretty good heuristic so I think it worth to remain it here after updating to vue 3 and another modern infinity scroller)
                if (this.items.length <= pageSize * 2 && !this.getRouteHash()) {
                    this.scrollDown();
                } else {
                    this.resetVariables();
                    this.reloadItems();
                }
                this.clearRouteHash();
                this.hash = null;
            },
            scrollDown() {
                Vue.nextTick(() => {
                    console.log("Scrolling down myDiv.scrollTop", this.scrollerDiv.scrollTop, "myDiv.scrollHeight", this.scrollerDiv.scrollHeight);
                    this.scrollerDiv.scrollTop = this.scrollerDiv.scrollHeight;
                });
            },
            isScrolledToBottom() {
                if (this.scrollerDiv) {
                    return this.scrollerDiv.scrollHeight - this.scrollerDiv.scrollTop - this.scrollerDiv.clientHeight < scrollingThreshold
                } else {
                    return false
                }
            },
            getInfo() {
                return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
                    console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
                    this.$store.commit(SET_TITLE, data.name);
                    this.$store.commit(SET_CHAT_USERS_COUNT, data.participantsCount);
                    this.$store.commit(SET_CHAT_ID, this.chatId);
                    this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, data.canEdit);
                    this.$store.commit(SET_CAN_BROADCAST_TEXT_MESSAGE, data.canBroadcast);
                    this.chatDto = data;
                }).catch(reason => {
                    if (reason.response.status == 404) {
                        this.goToChatList();
                    }
                }).then(() => {
                    const chatId = this.chatId;
                    if (chatId) {
                        axios.get(`/api/video/${chatId}/users`)
                            .then(response => response.data)
                            .then(data => {
                                bus.$emit(VIDEO_CALL_USER_COUNT_CHANGED, data);
                                this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, data.usersCount);
                            });
                    }
                });
            },
            goToChatList() {
                this.$router.push(({name: chat_list_name}))
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.chatDto = dto;
                    this.$store.commit(SET_CHAT_USERS_COUNT, this.chatDto.participantsCount);
                    this.$store.commit(SET_TITLE, this.chatDto.name);
                }
            },
            onChatDelete(dto) {
                if (dto.id == this.chatId) {
                    this.$router.push(({name: chat_list_name}))
                }
            },
            onUserProfileChanged(user) {
                this.items.forEach(item => {
                    if (item.owner.id == user.id) {
                        item.owner = user;
                    }
                });
            },
            onProfileSet() {
                this.getInfo();
                this.graphQlSubscribe();
                this.updateVideoRecordingState();
                this.setHashVariables();

                if (this.items.length === 0) {
                    this.reloadItems();
                }
            },
            onLoggedOut() {
                this.graphQlUnsubscribe();
                this.resetVariables();
            },

            onWsRestoredRefresh() {
                this.resetVariables();
                // Reset direction in order to fix bug when user relogin and after press button "update" all messages disappears due to non-initial direction.
                this.getInfo();
                this.reloadItems();
            },
            resetVariables() {
                this.aDirection = directionTop;
                this.items = [];
                this.startingFromItemId = null;
            },
            onVideoCallChanged(dto) {
                if (dto.chatId == this.chatId) {
                    this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, dto.usersCount);
                }
            },
            onResizedListener() {
                const isScrolled = this.isScrolledToBottom();
                if (isScrolled) {
                    this.scrollDown();
                }
            },
            openNewMessageDialog() { // on mobile OPEN_EDIT_MESSAGE with the null argument
                bus.$emit(OPEN_EDIT_MESSAGE, null);
            },

            onUserTyping(data) {
                console.debug("OnUserTyping", data);

                if (this.currentUser && this.currentUser.id == data.participantId) {
                    console.log("Skipping myself typing notifications");
                    return;
                }
                this.showTooltip = true;

                const idx = this.writingUsers.findIndex(value => value.login === data.login);
                if (idx !== -1) {
                    this.writingUsers[idx].timestamp = + new Date();
                } else {
                    this.writingUsers.push({timestamp: +new Date(), login: data.login})
                }
            },
            onUserBroadcast(dto) {
                console.log("onUserBroadcast", dto);
                const stripped = dto.text;
                if (stripped && stripped.length > 0) {
                    this.tooltipKey++;
                    this.showTooltip = true;
                    this.broadcastMessage = dto.text;
                } else {
                    this.broadcastMessage = null;
                }
            },
            getGraphQlSubscriptionQuery() {
                return `
                                subscription{
                                  chatEvents(chatId: ${this.chatId}) {
                                    eventType
                                    messageEvent {
                                      id
                                      text
                                      chatId
                                      ownerId
                                      createDateTime
                                      editDateTime
                                      owner {
                                        id
                                        login
                                        avatar
                                      }
                                      canEdit
                                      canDelete
                                      fileItemUuid
                                    }
                                    messageDeletedEvent {
                                      id
                                      chatId
                                    }
                                    userTypingEvent {
                                      login
                                      participantId
                                    }
                                    messageBroadcastEvent {
                                      login
                                      userId
                                      text
                                    }
                                    fileUploadedEvent {
                                      url
                                      previewUrl
                                      aType
                                      correlationId
                                    }
                                    participantsEvent {
                                      id
                                      login
                                      avatar
                                      admin
                                    }
                                  }
                                }
                `
            },
            onNextSubscriptionElement(e) {
                if (getChatEventsData(e).eventType === 'message_created') {
                    const d = getChatEventsData(e).messageEvent;
                    bus.$emit(MESSAGE_ADD, d);
                } else if (getChatEventsData(e).eventType === 'message_deleted') {
                    const d = getChatEventsData(e).messageDeletedEvent;
                    bus.$emit(MESSAGE_DELETED, d);
                } else if (getChatEventsData(e).eventType === 'message_edited') {
                    const d = getChatEventsData(e).messageEvent;
                    bus.$emit(MESSAGE_EDITED, d);
                } else if (getChatEventsData(e).eventType === "user_typing") {
                    const d = getChatEventsData(e).userTypingEvent;
                    bus.$emit(USER_TYPING, d);
                } else if (getChatEventsData(e).eventType === "user_broadcast") {
                    const d = getChatEventsData(e).messageBroadcastEvent;
                    bus.$emit(MESSAGE_BROADCAST, d);
                } else if (getChatEventsData(e).eventType === "file_uploaded") {
                    const d = getChatEventsData(e).fileUploadedEvent;
                    bus.$emit(FILE_UPLOADED, d);
                } else if (getChatEventsData(e).eventType === "participant_added") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_ADDED, d);
                } else if (getChatEventsData(e).eventType === "participant_deleted") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_DELETED, d);
                } else if (getChatEventsData(e).eventType === "participant_edited") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_EDITED, d);
                }
            },
            updateVideoRecordingState() {
                axios.get(`/api/video/${this.chatId}/record/status`).then(({data}) => {
                    this.$store.commit(SET_CAN_MAKE_RECORD, data.canMakeRecord);
                    if (data.canMakeRecord) {
                        const record = data.recordInProcess;
                        if (record) {
                            this.$store.commit(SET_SHOW_RECORD_STOP_BUTTON, true);
                        }
                    }
                })
            },
            setHashVariables() {
                this.hash = this.getRouteHash();
                this.highlightMessageId = this.getMessageId(this.hash);
            },
        },
        created() {
            this.searchStringChanged = debounce(this.searchStringChanged, 700, {leading:false, trailing:true});

            this.onResizedListener = debounce(this.onResizedListener, 100, {leading:true, trailing:true});
            this.onPanelResized = debounce(this.onPanelResized, 100, {leading:true, trailing:true});

            this.onScroll = throttle(this.onScroll, 400, {leading:true, trailing:true});

            this.initQueryAndWatcher();

            this.setHashVariables();
        },
        mounted() {
            window.addEventListener('resize', this.onResizedListener);

            this.$store.commit(SET_TITLE, `Chat #${this.chatId}`);
            this.$store.commit(SET_CHAT_USERS_COUNT, 0);
            this.$store.commit(SET_SHOW_SEARCH, true);
            this.$store.commit(SET_CHAT_ID, this.chatId);
            this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);
            this.$store.commit(SET_SEARCH_NAME, this.$vuetify.lang.t('$vuetify.search_in_messages'));

            // we trigger actions on load if profile was set
            if (this.currentUser) {
                this.onProfileSet();
            } // else we rely on PROFILE_SET

            this.$store.commit(SET_SHOW_CALL_BUTTON, true);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);

            bus.$on(MESSAGE_ADD, this.onNewMessage);
            bus.$on(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(MESSAGE_EDITED, this.onEditMessage);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(PROFILE_SET, this.onProfileSet);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
            bus.$on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);

            bus.$on(USER_TYPING, this.onUserTyping);
            bus.$on(MESSAGE_BROADCAST, this.onUserBroadcast);

            writingUsersTimerId = setInterval(()=>{
                const curr = + new Date();
                this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
            }, 500);

            this.scrollerDiv = document.getElementById("messagesScroller");
        },
        beforeDestroy() {
            this.graphQlUnsubscribe();
            window.removeEventListener('resize', this.onResizedListener);

            this.$store.commit(SET_SEARCH_NAME, null);

            bus.$off(MESSAGE_ADD, this.onNewMessage);
            bus.$off(MESSAGE_DELETED, this.onDeleteMessage);
            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(MESSAGE_EDITED, this.onEditMessage);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(PROFILE_SET, this.onProfileSet);
            bus.$off(LOGGED_OUT, this.onLoggedOut);
            bus.$off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);

            bus.$off(USER_TYPING, this.onUserTyping);
            bus.$off(MESSAGE_BROADCAST, this.onUserBroadcast);

            clearInterval(writingUsersTimerId);

            this.closeQueryWatcher();
        },
        destroyed() {
            this.$store.commit(SET_SHOW_CALL_BUTTON, false);
            this.$store.commit(SET_SHOW_HANG_BUTTON, false);
            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, 0);
            this.$store.commit(SET_CAN_BROADCAST_TEXT_MESSAGE, false);
            this.$store.commit(SET_SHOW_RECORD_START_BUTTON, false);
            this.$store.commit(SET_SHOW_RECORD_STOP_BUTTON, false);
        },
        components: {
            InfiniteLoading,
            MessageEdit,
            ChatVideo,
            Splitpanes, Pane,
            MessageItem
        },
        watch: {
            '$route': {
                // reacts on user manually typing hash - in this case we may trigger reload if we don't have the necessary message
                handler: function(newRoute, oldRoute) {
                    console.debug("Watched on newRoute", newRoute, " oldRoute", oldRoute);
                    if (newRoute.name === chat_name) {
                        this.setHashVariables();
                        if (this.hasHash) {
                            if (findIndexNonStrictly(this.items, {id: this.highlightMessageId}) === -1) {
                                this.resetVariables();
                                this.reloadItems(); // resets hash in infiniteHandler
                            } else {
                                this.hash = null; // reset cached hash explicitly in order not to subsequently use it in case when we have hash in scrolled to bottom
                            }
                        }
                    }
                },
                immediate: true,
                deep: true
            },
            '$vuetify.lang.current': {
                handler: function (newValue, oldValue) {
                    this.$store.commit(SET_SEARCH_NAME, this.$vuetify.lang.t('$vuetify.search_in_messages'));
                },
            },
        }
    }
</script>

<style scoped lang="stylus">
    @import "common.styl"

    #chatViewContainer {
        position: relative
        height: calc(100vh - 48px)
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
@import "common.styl"

$dot-size = 2px;
$dot-space = 4px;
$bg-color = $messageSelectedBackground;
$dot-color = darkgrey;

.splitpanes {background-color: #f8f8f8;}

.splitpanes__splitter {background-color: #ccc;position: relative; cursor: ns-resize}
.splitpanes__splitter:before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    transition: opacity 0.1s;

    // https://www.w3resource.com/html-css-exercise/html-css-practical-exercises/html-css-practical-exercise-28.php
    background-color: $bg-color;
    background-image: radial-gradient($bg-color 20%, transparent 40%), radial-gradient($dot-color 20%, transparent 40%);
    background-size: $dot-space $dot-space;
    background-position: 0 0, $dot-size $dot-size;
    background-repeat: repeat;

    opacity: 0;
    z-index: 1;
}
.splitpanes__splitter:hover:before {opacity: 1;}
.splitpanes--vertical > .splitpanes__splitter:before {left: -10px;right: -10px;height: 100%;}
.splitpanes--horizontal > .splitpanes__splitter:before {top: -10px;bottom: -10px;width: 100%;}
.panes3 {
    .splitpanes__splitter:nth-child(2):before {top: 0;bottom: -20px;width: 100%;}
    .splitpanes__splitter:nth-child(4):before {top: -20px;bottom: 0;width: 100%;}
}
.panes2 {
    .splitpanes__splitter:before {top: -20px;bottom: 0;width: 100%;}
}
</style>
