<template>
    <v-container class="ma-0 pa-0" id="chatViewContainer" fluid>
        <splitpanes ref="splOuter" class="default-theme" style="height: 100%"
                    :dbl-click-splitter="false"
                    @pane-add="onPanelAdd()" @pane-remove="onPanelRemove()" @resize="onPanelResized()">

            <pane v-bind:size="editAndMessagesSize">
                <splitpanes ref="splInner" class="default-theme" horizontal @pane-add="onPanelAdd()" @pane-remove="onPanelRemove()"  @resize="onPanelResized()">
                    <pane v-if="videoIsOnTop() && isAllowedVideo()" id="videoBlock" min-size="15" v-bind:size="videoSize">
                        <ChatVideo :chatDto="chatDto" :videoIsOnTop="videoIsOnTop()" />
                    </pane>

                    <pane v-bind:size="messagesSize">

                        <div v-if="pinnedPromoted" class="pinned-promoted">
                            <v-alert
                                :key="pinnedPromotedKey"
                                dense
                                color="red lighten-2"
                                dark
                                dismissible
                                prominent
                            >
                                <router-link :to="getPinnedRouteObject(pinnedPromoted)" style="text-decoration: none; color: white; cursor: pointer" v-html="pinnedPromoted.text">
                                </router-link>
                            </v-alert>
                        </div>

                        <MessageList ref="messageListRef" :chatDto="chatDto"/>
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
                    <pane max-size="70" min-size="8" v-bind:size="editSize" v-if="!isMobile()">
                        <MessageEdit :chatId="chatId"/>
                    </pane>
                </splitpanes>
            </pane>
            <pane v-if="videoIsAtSide() && isAllowedVideo()" id="videoBlock" min-size="15" v-bind:size="videoSize">
                <ChatVideo :chatDto="chatDto" :videoIsOnTop="videoIsOnTop()"/>
            </pane>

        </splitpanes>

        <v-speed-dial
            v-model="fab"
            v-if="isMobile()"
            :bottom="true"
            :right="true"
            direction="top"
            fixed
        >
            <template v-slot:activator>
                <v-btn
                    v-model="fab"
                    color="blue darken-2"
                    dark
                    fab
                >
                    <v-icon v-if="fab">
                        mdi-close
                    </v-icon>
                    <v-icon v-else>
                        mdi-plus
                    </v-icon>
                </v-btn>
            </template>
            <v-btn
                fab
                color="primary"
                small
                @click="openNewMessageDialog()"
            >
                <v-icon>mdi-message-plus-outline</v-icon>
            </v-btn>
            <v-btn
                fab
                color="success"
                small
                @click="copyCallLink()"
            >
                <v-icon>mdi-content-copy</v-icon>
            </v-btn>

            <v-btn fab v-if="showHangButton" small @click="addVideoSource()">
                <v-icon>mdi-video-plus</v-icon>
            </v-btn>
            <v-btn fab v-if="showRecordStartButton" small @click="startRecord()" :loading="initializingStaringVideoRecord">
                <v-icon>mdi-record-rec</v-icon>
            </v-btn>
            <v-btn fab v-if="showRecordStopButton" small @click="stopRecord()" :loading="initializingStoppingVideoRecord">
                <v-icon color="red">mdi-stop</v-icon>
            </v-btn>

            <v-btn
                v-if="!isScrolledToBottom()"
                color="primary"
                fab
                dark
                small
                @click="onClickScrollDown()"
            >
                <v-icon>mdi-chevron-down</v-icon>
            </v-btn>

        </v-speed-dial>

        <v-tooltip v-if="writingUsers.length || broadcastMessage" :activator="'#chatViewContainer'" bottom v-model="showTooltip" :key="tooltipKey">
            <span v-if="!broadcastMessage">{{writingUsers.map(v=>v.login).join(', ')}} {{ $vuetify.lang.t('$vuetify.user_is_writing') }}</span>
            <span v-else>{{broadcastMessage}}</span>
        </v-tooltip>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
        CHAT_DELETED,
        CHAT_EDITED,
        MESSAGE_ADD,
        MESSAGE_DELETED,
        MESSAGE_EDITED,
        USER_TYPING,
        USER_PROFILE_CHANGED,
        LOGGED_OUT,
        VIDEO_CALL_USER_COUNT_CHANGED,
        MESSAGE_BROADCAST,
        REFRESH_ON_WEBSOCKET_RESTORED,
        OPEN_EDIT_MESSAGE,
        PROFILE_SET,
        PREVIEW_CREATED,
        PARTICIPANT_ADDED,
        PARTICIPANT_DELETED,
        PARTICIPANT_EDITED,
        VIDEO_DIAL_STATUS_CHANGED,
        PINNED_MESSAGE_PROMOTED,
        PINNED_MESSAGE_UNPROMOTED, ADD_VIDEO_SOURCE_DIALOG, FILE_CREATED, FILE_REMOVED,
    } from "./bus";
    import {chat_list_name, chat_name, messageIdHashPrefix, videochat_name} from "./routes";
    import MessageEdit from "./MessageEdit";
    import ChatVideo from "./ChatVideo";
    import MessageList from "./MessageList";

    import {mapGetters} from "vuex";

    import {
        GET_INITIALIZING_STARTING_VIDEO_RECORD, GET_INITIALIZING_STOPPING_VIDEO_RECORD,
        GET_SHOW_HANG_BUTTON, GET_SHOW_RECORD_START_BUTTON, GET_SHOW_RECORD_STOP_BUTTON,
        GET_USER, SET_AVATAR,
        SET_CAN_BROADCAST_TEXT_MESSAGE,
        SET_CAN_MAKE_RECORD,
        SET_CHAT_ID,
        SET_CHAT_USERS_COUNT, SET_INITIALIZING_STARTING_VIDEO_RECORD, SET_INITIALIZING_STOPPING_VIDEO_RECORD,
        SET_SEARCH_NAME, SET_SHOULD_PHONE_BLINK,
        SET_SHOW_CALL_BUTTON,
        SET_SHOW_CHAT_EDIT_BUTTON,
        SET_SHOW_HANG_BUTTON,
        SET_SHOW_RECORD_START_BUTTON,
        SET_SHOW_RECORD_STOP_BUTTON,
        SET_SHOW_SEARCH, SET_TET_A_TET,
        SET_TITLE,
        SET_VIDEO_CHAT_USERS_COUNT
    } from "./store";
    import { Splitpanes, Pane } from 'splitpanes'
    import debounce from "lodash/debounce";
    import graphqlSubscriptionMixin from "./graphqlSubscriptionMixin"
    import 'splitpanes/dist/splitpanes.css';
    import {
        getStoredVideoPosition,
        VIDEO_POSITION_AUTO,
        VIDEO_POSITION_ON_THE_TOP,
        VIDEO_POSITION_SIDE
    } from "@/localStore";
    import {copyCallLink, offerToJoinToPublicChatStatus} from "@/utils";

    const KEY_DESKTOP_TOP_WITH_VIDEO_PANELS = 'desktopTopWithVideo2';
    const KEY_DESKTOP_TOP_WITHOUT_VIDEO_PANELS = 'desktopTopWithoutVideo2'
    const KEY_DESKTOP_SIDE_WITH_VIDEO_PANELS = 'desktopSideWithVideo2';
    const KEY_DESKTOP_SIDE_WITHOUT_VIDEO_PANELS = 'desktopSideWithoutVideo2'
    const KEY_MOBILE_TOP_WITH_VIDEO_PANELS = 'mobileTopWithVideo2';
    const KEY_MOBILE_TOP_WITHOUT_VIDEO_PANELS = 'mobileTopWithoutVideo2'
    const KEY_MOBILE_SIDE_WITH_VIDEO_PANELS = 'mobileSideWithVideo2';
    const KEY_MOBILE_SIDE_WITHOUT_VIDEO_PANELS = 'mobileSideWithoutVideo2'


    let writingUsersTimerId;

    const getChatEventsData = (message) => {
        return message.data?.chatEvents
    };

    const chatDtoFactory = () => {
        return {
            participantIds:[],
            participants:[],
        }
    }

    const emptyStoredPanes = (key) => {
        switch (key) {
            case KEY_DESKTOP_TOP_WITH_VIDEO_PANELS:
                return {editAndMessages:100, messages:50, edit:20, video: 30}
            case KEY_DESKTOP_TOP_WITHOUT_VIDEO_PANELS:
                return {editAndMessages:100, messages:70, edit:30}
            case KEY_DESKTOP_SIDE_WITH_VIDEO_PANELS:
                return {editAndMessages:50, messages:60, edit:40, video: 50}
            case KEY_DESKTOP_SIDE_WITHOUT_VIDEO_PANELS:
                return {editAndMessages:100, messages:80, edit:20}
            case KEY_MOBILE_TOP_WITH_VIDEO_PANELS:
                return {editAndMessages:100, messages:60, video:40}
            case KEY_MOBILE_TOP_WITHOUT_VIDEO_PANELS:
                return {editAndMessages:100, messages:100}
            case KEY_MOBILE_SIDE_WITH_VIDEO_PANELS:
                return {editAndMessages:100, messages:100, video:40}
            case KEY_MOBILE_SIDE_WITHOUT_VIDEO_PANELS:
                return {editAndMessages:100, messages:100}

        }
        console.warn("Not found default panel sizes")
        return {}
    }

    export default {
        mixins: [
            graphqlSubscriptionMixin('chatEvents')
        ],
        data() {
            return {
                chatDto: chatDtoFactory(),
                writingUsers: [],
                showTooltip: true,
                broadcastMessage: null,
                tooltipKey: 0,
                pinnedPromoted: null,
                pinnedPromotedKey: +new Date(),
                fab: false
            }
        },
        computed: {
            chatId() {
                return this.$route.params.id
            },
            ...mapGetters({currentUser: GET_USER}),

            videoSize() {
                const stored = this.readFromStore();
                return stored.video;
            },
            messagesSize() {
                const stored = this.readFromStore();
                return stored.messages;
            },
            editSize() {
                // not need here because it's not used in mobile
                const stored = this.getStored();
                return stored.edit;
            },
            editAndMessagesSize() {
                const stored = this.getStored();
                return stored.editAndMessages;
            },
            ...mapGetters({
                showHangButton: GET_SHOW_HANG_BUTTON,
                showRecordStartButton: GET_SHOW_RECORD_START_BUTTON,
                showRecordStopButton: GET_SHOW_RECORD_STOP_BUTTON,
                initializingStaringVideoRecord: GET_INITIALIZING_STARTING_VIDEO_RECORD,
                initializingStoppingVideoRecord: GET_INITIALIZING_STOPPING_VIDEO_RECORD,
            }),
        },
        methods: {
            getStored() {
                let keyWithVideo;
                let keyWithoutVideo;
                if (!this.isMobile()) {
                    if (this.videoIsOnTop()) {
                        keyWithVideo = KEY_DESKTOP_TOP_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_DESKTOP_TOP_WITHOUT_VIDEO_PANELS;
                    } else {
                        keyWithVideo = KEY_DESKTOP_SIDE_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_DESKTOP_SIDE_WITHOUT_VIDEO_PANELS;
                    }
                } else {
                    if (this.videoIsOnTop()) {
                        keyWithVideo = KEY_MOBILE_TOP_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_MOBILE_TOP_WITHOUT_VIDEO_PANELS;
                    } else {
                        keyWithVideo = KEY_MOBILE_SIDE_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_MOBILE_SIDE_WITHOUT_VIDEO_PANELS;
                    }
                }

                const key = this.isAllowedVideo() ? keyWithVideo : keyWithoutVideo;
                const mbItem = localStorage.getItem(key);
                if (!mbItem) {
                    return emptyStoredPanes(key);
                } else {
                    return JSON.parse(mbItem);
                }
            },
            saveToStored(obj) {
                let keyWithVideo;
                let keyWithoutVideo;
                if (!this.isMobile()) {
                    if (this.videoIsOnTop()) {
                        keyWithVideo = KEY_DESKTOP_TOP_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_DESKTOP_TOP_WITHOUT_VIDEO_PANELS;
                    } else {
                        keyWithVideo = KEY_DESKTOP_SIDE_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_DESKTOP_SIDE_WITHOUT_VIDEO_PANELS;
                    }
                } else {
                    if (this.videoIsOnTop()) {
                        keyWithVideo = KEY_MOBILE_TOP_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_MOBILE_TOP_WITHOUT_VIDEO_PANELS;
                    } else {
                        keyWithVideo = KEY_MOBILE_SIDE_WITH_VIDEO_PANELS;
                        keyWithoutVideo = KEY_MOBILE_SIDE_WITHOUT_VIDEO_PANELS;
                    }
                }

                if (this.isAllowedVideo()) {
                    localStorage.setItem(keyWithVideo, JSON.stringify(obj));
                } else {
                    localStorage.setItem(keyWithoutVideo, JSON.stringify(obj));
                }
            },


            onPanelAdd() {
                console.log("On panel add", this.$refs.splOuter.panes);
                this.$nextTick(() => {
                    const stored = this.getStored();
                    this.restorePanelsSize(stored);
                })

            },
            onPanelRemove() {
                console.log("On panel removed", this.$refs.splOuter.panes);
                this.$nextTick(() => {
                    const stored = this.getStored();
                    this.restorePanelsSize(stored);
                })
            },

            onPanelResized() {
                this.saveToStored(this.prepareForStore());
            },

            readFromStore() {
                let stored = this.getStored();
                if (!stored) {
                    this.saveToStored(emptyStoredPanes())
                    stored = this.getStored();
                }
                return stored
            },
            prepareForStore() {
                const outerPaneSizes = this.$refs.splOuter.panes.map(i => i.size);
                const innerPaneSizes = this.$refs.splInner.panes.map(i => i.size);
                if (this.videoIsOnTop()) {
                    if (innerPaneSizes.length == 3) {
                        return {
                            video: innerPaneSizes[0],
                            messages: innerPaneSizes[1],
                            edit: innerPaneSizes[2]
                        };
                    } else {
                        return {
                            messages: innerPaneSizes[0],
                            edit: innerPaneSizes[1]
                        };
                    }
                } else { // side
                    const ret = {
                        editAndMessages: outerPaneSizes[0],
                        messages: innerPaneSizes[0],
                        edit: innerPaneSizes[1]
                    };
                    if (outerPaneSizes[1]) {
                        ret.video = outerPaneSizes[1];
                    }
                    return ret
                }
            },
            restorePanelsSize(stored) {
                console.log("Restoring from", stored);
                if (this.videoIsOnTop()) {
                    if (this.$refs.splInner.panes.length == 3) {
                        this.$refs.splInner.panes[0].size = stored.video;
                        if (this.$refs.splInner.panes[1]) {
                            this.$refs.splInner.panes[1].size = stored.messages;
                        }
                        if (this.$refs.splInner.panes[2]) {
                            this.$refs.splInner.panes[2].size = stored.edit;
                        }
                    } else {
                        this.$refs.splInner.panes[0].size = stored.messages;
                        if (this.$refs.splInner.panes[1]) {
                            this.$refs.splInner.panes[1].size = stored.edit;
                        }
                    }
                } else { // side
                    if (this.$refs.splOuter) {
                        this.$refs.splOuter.panes[0].size = stored.editAndMessages;
                        if (this.$refs.splOuter.panes[1]) {
                            this.$refs.splOuter.panes[1].size = stored.video;
                        }
                    }
                    if (this.$refs.splInner) {
                        this.$refs.splInner.panes[0].size = stored.messages;
                        if (this.$refs.splInner.panes[1]) {
                            this.$refs.splInner.panes[1].size = stored.edit;
                        }
                    }
                }
            },

            isAllowedVideo() {
                return this.currentUser && this.$router.currentRoute.name == videochat_name && this.chatDto && this.chatDto.participantIds && this.chatDto.participantIds.length
            },

            videoIsOnTop() {
                const stored = getStoredVideoPosition();
                if (stored == VIDEO_POSITION_AUTO) {
                    return this.isMobile()
                } else {
                    return getStoredVideoPosition() == VIDEO_POSITION_ON_THE_TOP;
                }
            },

            videoIsAtSide() {
                return !this.videoIsOnTop();
            },

            fetchAndSetChat() {
                return axios.get(`/api/chat/${this.chatId}`).then(({data}) => {
                    console.log("Got info about chat in ChatView, chatId=", this.chatId, data);
                    this.$store.commit(SET_TITLE, data.name);
                    this.$store.commit(SET_AVATAR, data.avatar);
                    this.$store.commit(SET_CHAT_USERS_COUNT, data.participantsCount);
                    this.$store.commit(SET_CHAT_ID, this.chatId);
                    this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, data.canEdit);
                    this.$store.commit(SET_CAN_BROADCAST_TEXT_MESSAGE, data.canBroadcast);
                    this.$store.commit(SET_TET_A_TET, data.tetATet);
                    this.chatDto = data;
                })
            },
            getInfo() {
                return this.fetchAndSetChat().catch(reason => {
                    if (reason.response.status == 404) {
                        this.goToChatList();
                        return Promise.reject();
                    } else if (reason.response.status == offerToJoinToPublicChatStatus) {
                        return axios.put(`/api/chat/${this.chatId}/join`).then(()=>{
                            return this.fetchAndSetChat();
                        })
                    } else {
                        return Promise.resolve();
                    }
                }).then(() => {
                    // async call
                    axios.get(`/api/video/${this.chatId}/users`)
                        .then(response => response.data)
                        .then(data => {
                            bus.$emit(VIDEO_CALL_USER_COUNT_CHANGED, data);
                            this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, data.usersCount);
                        })
                    this.fetchPromotedMessage();
                    return Promise.resolve();
                })
            },
            goToChatList() {
                this.$router.push(({name: chat_list_name}))
            },
            onChatChange(dto) {
                if (dto.id == this.chatId) {
                    this.chatDto = dto;
                    this.$store.commit(SET_CHAT_USERS_COUNT, this.chatDto.participantsCount);
                    this.$store.commit(SET_TITLE, this.chatDto.name);
                    this.$store.commit(SET_AVATAR, this.chatDto.avatar);
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
                this.getInfo().then(() => {
                    this.graphQlSubscribe();
                    return this.updateVideoRecordingState();
                }).then(() => {
                    this.$refs.messageListRef.setHashVariables();
                });
            },
            onLoggedOut() {
                this.graphQlUnsubscribe();
                this.$refs.messageListRef.resetVariables();
            },

            onWsRestoredRefresh() {
                this.$refs.messageListRef.resetVariables();
                // Reset direction in order to fix bug when user relogin and after press button "update" all messages disappears due to non-initial direction.
                this.getInfo().then(()=>{
                    this.$refs.messageListRef.reloadItems();
                });
            },
            onVideoCallChanged(dto) {
                if (dto.chatId == this.chatId) {
                    this.$store.commit(SET_VIDEO_CHAT_USERS_COUNT, dto.usersCount);
                }
            },
            openNewMessageDialog() { // on mobile OPEN_EDIT_MESSAGE with the null argument
                bus.$emit(OPEN_EDIT_MESSAGE, null);
            },
            copyCallLink(){
                copyCallLink(this.chatId)
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
                                fragment DisplayMessageDtoFragment on DisplayMessageDto {
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
                                    shortInfo
                                  }
                                  canEdit
                                  canDelete
                                  fileItemUuid
                                  embedMessage {
                                    id
                                    chatId
                                    chatName
                                    text
                                    owner {
                                      id
                                      login
                                      avatar
                                      shortInfo
                                    }
                                    embedType
                                    isParticipant
                                  }
                                  pinned
                                  blogPost
                                }

                                subscription{
                                  chatEvents(chatId: ${this.chatId}) {
                                    eventType
                                    messageEvent {
                                      ...DisplayMessageDtoFragment
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
                                    previewCreatedEvent {
                                      id
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
                                    promoteMessageEvent {
                                      totalCount
                                      message {
                                        ...DisplayMessageDtoFragment
                                      }
                                    }
                                    fileEvent {
                                      fileInfoDto {
                                        id
                                        filename
                                        url
                                        publicUrl
                                        previewUrl
                                        size
                                        canDelete
                                        canEdit
                                        canShare
                                        lastModified
                                        ownerId
                                        owner {
                                          id
                                          login
                                          avatar
                                        }
                                        canPlayAsVideo
                                        canShowAsImage
                                      }
                                      count
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
                } else if (getChatEventsData(e).eventType === "preview_created") {
                    const d = getChatEventsData(e).previewCreatedEvent;
                    bus.$emit(PREVIEW_CREATED, d);
                } else if (getChatEventsData(e).eventType === "participant_added") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_ADDED, d);
                } else if (getChatEventsData(e).eventType === "participant_deleted") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_DELETED, d);
                } else if (getChatEventsData(e).eventType === "participant_edited") {
                    const d = getChatEventsData(e).participantsEvent;
                    bus.$emit(PARTICIPANT_EDITED, d);
                } else if (getChatEventsData(e).eventType === "pinned_message_promote") {
                    const d = getChatEventsData(e).promoteMessageEvent;
                    bus.$emit(PINNED_MESSAGE_PROMOTED, d);
                } else if (getChatEventsData(e).eventType === "pinned_message_unpromote") {
                    const d = getChatEventsData(e).promoteMessageEvent;
                    bus.$emit(PINNED_MESSAGE_UNPROMOTED, d);
                } else if (getChatEventsData(e).eventType === "file_created") {
                    const d = getChatEventsData(e).fileEvent;
                    bus.$emit(FILE_CREATED, d);
                } else if (getChatEventsData(e).eventType === "file_removed") {
                    const d = getChatEventsData(e).fileEvent;
                    bus.$emit(FILE_REMOVED, d);
                }
            },
            updateVideoRecordingState() {
                return axios.get(`/api/video/${this.chatId}/record/status`).then(({data}) => {
                    this.$store.commit(SET_CAN_MAKE_RECORD, data.canMakeRecord);
                    if (data.canMakeRecord) {
                        const record = data.recordInProcess;
                        if (record) {
                            this.$store.commit(SET_SHOW_RECORD_STOP_BUTTON, true);
                        }
                    }
                })
            },
            onChatDialStatusChange(dto) {
                if (this.chatDto.tetATet) {
                    for (const videoDialChanged of dto.dials) {
                        if (this.currentUser.id != videoDialChanged.userId) {
                            this.$store.commit(SET_SHOULD_PHONE_BLINK, videoDialChanged.status);
                        }
                    }
                }
            },
            isScrolledToBottom() {
                return this.$refs.messageListRef?.isScrolledToBottom();
            },
            onClickScrollDown() {
                return this.$refs.messageListRef.onClickScrollDown();
            },
            fetchPromotedMessage() {
                axios.get(`/api/chat/${this.chatId}/message/pin/promoted`).then((response) => {
                    if (response.status != 204) {
                        this.pinnedPromoted = response.data;
                    }
                });
            },
            onPinnedMessagePromoted(item) {
                this.pinnedPromoted = item.message;
                this.pinnedPromotedKey++;
            },
            onPinnedMessageUnpromoted(item) {
                if (this.pinnedPromoted && this.pinnedPromoted.id == item.message.id) {
                    this.pinnedPromoted = null;
                }
            },
            getPinnedRouteObject(item) {
                const routeName = this.isVideoRoute() ? videochat_name : chat_name;
                return {name: routeName, params: {id: item.chatId}, hash: messageIdHashPrefix + item.id};
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            addVideoSource() {
                bus.$emit(ADD_VIDEO_SOURCE_DIALOG);
            },
            startRecord() {
                axios.put(`/api/video/${this.chatId}/record/start`);
                this.$store.commit(SET_INITIALIZING_STARTING_VIDEO_RECORD, true)
            },
            stopRecord() {
                axios.put(`/api/video/${this.chatId}/record/stop`);
                this.$store.commit(SET_INITIALIZING_STOPPING_VIDEO_RECORD, true)
            },
        },
        created() {
            this.onPanelResized = debounce(this.onPanelResized, 100, {leading:true, trailing:true});
        },
        mounted() {

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

            bus.$on(CHAT_EDITED, this.onChatChange);
            bus.$on(CHAT_DELETED, this.onChatDelete);
            bus.$on(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$on(PROFILE_SET, this.onProfileSet);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
            bus.$on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);

            bus.$on(USER_TYPING, this.onUserTyping);
            bus.$on(MESSAGE_BROADCAST, this.onUserBroadcast);
            bus.$on(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
            bus.$on(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
            bus.$on(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);

            writingUsersTimerId = setInterval(()=>{
                const curr = + new Date();
                this.writingUsers = this.writingUsers.filter(value => (value.timestamp + 1*1000) > curr);
            }, 500);

        },
        beforeDestroy() {
            this.graphQlUnsubscribe();

            this.$store.commit(SET_SEARCH_NAME, null);

            bus.$off(CHAT_EDITED, this.onChatChange);
            bus.$off(CHAT_DELETED, this.onChatDelete);
            bus.$off(USER_PROFILE_CHANGED, this.onUserProfileChanged);
            bus.$off(PROFILE_SET, this.onProfileSet);
            bus.$off(LOGGED_OUT, this.onLoggedOut);
            bus.$off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
            bus.$off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);

            bus.$off(USER_TYPING, this.onUserTyping);
            bus.$off(MESSAGE_BROADCAST, this.onUserBroadcast);
            bus.$off(VIDEO_DIAL_STATUS_CHANGED, this.onChatDialStatusChange);
            bus.$off(PINNED_MESSAGE_PROMOTED, this.onPinnedMessagePromoted);
            bus.$off(PINNED_MESSAGE_UNPROMOTED, this.onPinnedMessageUnpromoted);

            clearInterval(writingUsersTimerId);

            this.pinnedPromoted = null;
            this.pinnedPromotedKey = null;

            this.chatDto = chatDtoFactory();

            this.$store.commit(SET_AVATAR, null);

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
            MessageEdit,
            ChatVideo,
            Splitpanes, Pane,
            MessageList,
        },
        watch: {
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
        height $calculatedHeight
        //width: calc(100% - 80px)
    }
    //
    //@media screen and (max-width: $mobileWidth) {
    //    #chatViewContainer {
    //        height: calc(100vh - 116px)
    //    }
    //}

    .pinned-promoted {
        position: absolute;
        z-index: 4;
        width: 100%
    }

</style>
