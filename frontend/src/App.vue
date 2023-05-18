<template>
    <v-app>
        <!-- https://vuetifyjs.com/en/components/application/ -->
        <v-navigation-drawer
                left
                app
                :clipped="true"
                v-model="drawer"
        >
            <template v-slot:prepend>
                <v-list-item two-line v-if="currentUser" @click.prevent="onProfileClicked()" link :href="require('./routes').profile">
                    <v-list-item-avatar  v-if="currentUser.avatar">
                        <img :src="currentUserAvatar"/>
                    </v-list-item-avatar>

                    <v-list-item-content>
                        <v-list-item-title class="user-login">{{currentUser.login}}</v-list-item-title>
                        <v-list-item-subtitle v-if="showCurrentUserSubtitle()">{{currentUser.shortInfo}}</v-list-item-subtitle>
                    </v-list-item-content>
                </v-list-item>
            </template>

            <v-divider></v-divider>

            <v-list dense>
                <v-list-item @click.prevent="goHome()" :href="require('./routes').root">
                    <v-list-item-icon><v-icon>mdi-forum</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.chats') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click.prevent="goBlog()" :href="require('./routes').blog">
                    <v-list-item-icon><v-icon>mdi-postage-stamp</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.blogs') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="createChat()">
                    <v-list-item-icon><v-icon>mdi-plus</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title id="new-chat-dialog-button">{{ $vuetify.lang.t('$vuetify.new_chat') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="displayChatFiles()" v-if="shouldDisplayFiles()">
                    <v-list-item-icon><v-icon>mdi-file-download</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.files') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="openPinnedMessages()" v-if="shouldPinnedMessages()">
                    <v-list-item-icon><v-icon>mdi-pin</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.pinned_messages') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="findUser()">
                    <v-list-item-icon><v-icon>mdi-magnify</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.find_user') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="editChat()" v-if="shouldDisplayEditChat()">
                    <v-list-item-icon><v-icon>mdi-lead-pencil</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.edit_chat') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click.prevent="goProfile()" v-if="shouldDisplayProfile()" :href="require('./routes').profile">
                    <v-list-item-icon><v-icon>mdi-account</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.profile') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="openVideoSettings()" v-if="shouldDisplayVideoSettings()">
                    <v-list-item-icon><v-icon>mdi-cog</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.video_settings') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="openLocale()">
                    <v-list-item-icon><v-icon>mdi-flag</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.language') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="logout()" v-if="shouldDisplayLogout">
                    <v-list-item-icon><v-icon>mdi-logout</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.logout') }}</v-list-item-title></v-list-item-content>
                </v-list-item>
            </v-list>
        </v-navigation-drawer>


        <v-app-bar
                color='indigo'
                dark
                app
                id="myAppBar"
                :clipped-left="true"
                dense
        >
            <v-app-bar-nav-icon @click="toggleLeftNavigation"></v-app-bar-nav-icon>
            <template v-if="showSearchButton || !isMobile()">
                <v-badge v-if="showCallButton || showHangButton"
                    :content="videoChatUsersCount"
                    :value="videoChatUsersCount"
                    color="green"
                    overlap
                    offset-y="1.8em"
                >
                    <v-btn v-if="showCallButton" icon @click="createCall()" :title="tetATet ? $vuetify.lang.t('$vuetify.call_up') : $vuetify.lang.t('$vuetify.enter_into_call')">
                        <v-icon color="green">{{tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                    </v-btn>
                    <v-btn v-if="showHangButton" icon @click="stopCall()" :title="$vuetify.lang.t('$vuetify.leave_call')">
                        <v-icon :class="shouldPhoneBlink ? 'call-blink' : 'red--text'">mdi-phone</v-icon>
                    </v-btn>
                </v-badge>

                <template v-if="canShowMicrophoneButton">
                    <v-btn v-if="showHangButton && !isMobile() && showMicrophoneOnButton" icon @click="offMicrophone()" :title="$vuetify.lang.t('$vuetify.mute_audio')"><v-icon>mdi-microphone</v-icon></v-btn>
                    <v-btn v-if="showHangButton && !isMobile() && showMicrophoneOffButton" icon @click="onMicrophone()" :title="$vuetify.lang.t('$vuetify.unmute_audio')"><v-icon>mdi-microphone-off</v-icon></v-btn>
                </template>

                <v-btn v-if="(showCallButton || showHangButton) && !isMobile()" icon @click="copyCallLink()" :title="$vuetify.lang.t('$vuetify.copy_video_call_link')">
                    <v-icon>mdi-content-copy</v-icon>
                </v-btn>

                <v-btn v-if="showHangButton && !isMobile()" icon @click="addScreenSource()" :title="$vuetify.lang.t('$vuetify.screen_share')">
                    <v-icon>mdi-monitor-screenshot</v-icon>
                </v-btn>
                <v-btn v-if="showHangButton && !isMobile()" icon @click="addVideoSource()" :title="$vuetify.lang.t('$vuetify.source_add')">
                    <v-icon>mdi-video-plus</v-icon>
                </v-btn>
                <v-btn v-if="showRecordStartButton && !isMobile()" icon @click="startRecord()" :loading="initializingStaringVideoRecord" :title="$vuetify.lang.t('$vuetify.start_record')">
                    <v-icon>mdi-record-rec</v-icon>
                </v-btn>
                <v-btn v-if="showRecordStopButton && !isMobile()" icon @click="stopRecord()" :loading="initializingStoppingVideoRecord" :title="$vuetify.lang.t('$vuetify.stop_record')">
                    <v-icon color="red">mdi-stop</v-icon>
                </v-btn>
                <v-spacer></v-spacer>
                <img v-if="chatAvatar" class="v-avatar chat-avatar" :src="chatAvatar"/>
                <v-toolbar-title color="white" class="d-flex flex-column px-2 app-title" :class="chatId ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked" :style="{'cursor': chatId ? 'pointer' : 'default'}">
                    <div :class="!isMobile() ? ['align-self-center'] : []" class="app-title-text" v-html="title"></div>
                    <div v-if="chatUsersCount" :class="!isMobile() ? ['align-self-center'] : []" class="app-title-subtext">
                        {{ chatUsersCount }} {{ $vuetify.lang.t('$vuetify.participants') }}</div>
                </v-toolbar-title>
            </template>
            <v-spacer></v-spacer>

            <v-btn v-if="isShowSearch && showSearchButton && isMobile()" icon :title="searchName" @click="onOpenSearch()">
                <v-icon>{{ hasSearchString ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
            </v-btn>
            <v-card light v-if="isShowSearch && !showSearchButton || !isMobile()" :width="isMobile() ? '100%' : ''">
                <v-text-field :autofocus="isMobile()" prepend-icon="mdi-magnify" hide-details single-line @input="clearRouteHash()" v-model="searchString" :label="searchName" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" @blur="showSearchButton=true"></v-text-field>
            </v-card>

            <template v-if="showSearchButton || !isMobile()">
                <v-badge
                    :content="notificationsCount"
                    :value="notificationsCount"
                    color="red"
                    overlap
                    :offset-y="isMobile() ? '' : '1.8em'"
                >
                    <v-btn
                        :small="isMobile()"
                        icon :title="$vuetify.lang.t('$vuetify.notifications')"
                        @click="onNotificationsClicked()"
                    >
                        <v-icon>mdi-bell</v-icon>
                    </v-btn>
                </v-badge>
            </template>
        </v-app-bar>


        <v-main>
            <v-container fluid class="ma-0 pa-0" style="height: 100%">
                <v-snackbar v-model="showAlert" :color="errorColor" timeout="-1" :multi-line="true" :transition="false">
                    {{ lastError }}

                    <template v-slot:action="{ attrs }">
                        <v-btn
                            text
                            v-bind="attrs"
                            @click="closeError()"
                        >
                            Close
                        </v-btn>
                    </template>
                </v-snackbar>
                <v-snackbar v-model="showWebsocketRestored" color="black" timeout="-1" :multi-line="true" :transition="false">
                    {{ $vuetify.lang.t('$vuetify.websocket_restored') }}
                    <template v-slot:action="{ attrs }">
                        <v-btn
                            text
                            v-bind="attrs"
                            @click="onPressWebsocketRestored()"
                        >
                            {{ $vuetify.lang.t('$vuetify.btn_update') }}
                        </v-btn>
                        <v-btn text v-bind="attrs" @click="showWebsocketRestored = false">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>

                    </template>
                </v-snackbar>
                <v-snackbar v-model="invitedVideoChatAlert" color="success" timeout="-1" :multi-line="true" top :transition="false">
                    <span class="call-blink">
                        {{ $vuetify.lang.t('$vuetify.you_called', invitedVideoChatId, invitedVideoChatName) }}
                    </span>
                    <template v-slot:action="{ attrs }">
                        <v-btn icon v-bind="attrs" @click="onClickInvitation()"><v-icon color="white">mdi-phone</v-icon></v-btn>
                        <v-btn icon v-bind="attrs" @click="onClickCancelInvitation()"><v-icon color="white">mdi-close-circle</v-icon></v-btn>
                    </template>
                </v-snackbar>

                <LoginModal/>
                <ChatEditModal/>
                <ChatParticipantsModal/>
                <SimpleModal/>
                <PermissionsWarningModal/>
                <ChooseAvatarModal/>
                <FindUserModal/>
                <FileUploadModal/>
                <FileListModal/>
                <VideoGlobalSettingsModal/>
                <FileTextEditModal/>
                <LanguageModal/>
                <VideoAddNewSourceModal/>
                <MessageEditModal v-if="isMobile()"/>
                <MessageEditLinkModal/>
                <MessageEditColorModal/>
                <NotificationsModal/>
                <MessageEditMediaModal/>
                <MessageResendToModal/>
                <PinnedMessagesModal/>
                <PlayerModal/>

                <router-view :key="`routerView`+`${$route.params.id}`"/>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import 'typeface-roboto'
    import axios from 'axios';
    import LoginModal from "./LoginModal";
    import {mapGetters} from 'vuex'
    import {
        FETCH_USER_PROFILE,
        GET_CHAT_ID,
        GET_CHAT_USERS_COUNT,
        GET_ERROR_COLOR,
        GET_LAST_ERROR,
        GET_SHOW_ALERT,
        GET_SHOW_CALL_BUTTON,
        GET_SHOW_CHAT_EDIT_BUTTON,
        GET_SHOW_HANG_BUTTON,
        GET_SHOW_SEARCH,
        GET_TITLE,
        GET_AVATAR,
        GET_USER,
        GET_VIDEO_CHAT_USERS_COUNT,
        UNSET_USER,
        GET_SHOW_RECORD_START_BUTTON,
        GET_SHOW_RECORD_STOP_BUTTON,
        SET_SHOW_RECORD_START_BUTTON,
        SET_SHOW_RECORD_STOP_BUTTON,
        FETCH_NOTIFICATIONS,
        GET_NOTIFICATIONS,
        UNSET_NOTIFICATIONS,
        FETCH_AVAILABLE_OAUTH2_PROVIDERS,
        GET_SEARCH_NAME,
        GET_SHOULD_PHONE_BLINK,
        GET_TET_A_TET,
        GET_SHOW_MICROPHONE_ON_BUTTON,
        GET_SHOW_MICROPHONE_OFF_BUTTON,
        GET_CAN_SHOW_MICROPHONE_BUTTON,
        SET_TITLE,
        SET_INITIALIZING_STARTING_VIDEO_RECORD,
        SET_INITIALIZING_STOPPING_VIDEO_RECORD,
        GET_INITIALIZING_STARTING_VIDEO_RECORD,
        GET_INITIALIZING_STOPPING_VIDEO_RECORD
    } from "./store";
    import bus, {
        LOGGED_OUT,
        OPEN_CHAT_EDIT,
        OPEN_PARTICIPANTS_DIALOG,
        OPEN_PERMISSIONS_WARNING_MODAL,
        VIDEO_CALL_INVITED,
        REFRESH_ON_WEBSOCKET_RESTORED,
        OPEN_FIND_USER,
        OPEN_VIEW_FILES_DIALOG,
        OPEN_VIDEO_SETTINGS,
        OPEN_LANGUAGE_MODAL,
        ADD_VIDEO_SOURCE_DIALOG,
        ADD_SCREEN_SOURCE,
        VIDEO_RECORDING_CHANGED,
        OPEN_NOTIFICATIONS_DIALOG,
        PROFILE_SET,
        WEBSOCKET_RESTORED,
        OPEN_PINNED_MESSAGES_MODAL,
        VIDEO_OPENED,
        VIDEO_CLOSED,
        SET_LOCAL_MICROPHONE_MUTED,
    } from "./bus";
    import ChatEditModal from "./ChatEditModal";
    import {chat_name, profile_self_name, chat_list_name, videochat_name, blog} from "./routes";
    import SimpleModal from "./SimpleModal";
    import ChooseAvatarModal from "./ChooseAvatarModal";
    import ChatParticipantsModal from "./ChatParticipantsModal";
    import PermissionsWarningModal from "./PermissionsWarningModal";
    import FindUserModal from "./FindUserModal";
    import FileUploadModal from './FileUploadModal';
    import FileListModal from "./FileListModal";
    import VideoGlobalSettingsModal from './VideoGlobalSettingsModal';
    import FileTextEditModal from "./FileTextEditModal";
    import LanguageModal from "./LanguageModal";
    import VideoAddNewSourceModal from "@/VideoAddNewSourceModal";
    import MessageEditModal from "@/MessageEditModal";
    import MessageEditLinkModal from "@/MessageEditLinkModal";
    import MessageEditColorModal from "@/MessageEditColorModal";
    import NotificationsModal from "@/NotificationsModal";
    import MessageEditMediaModal from "@/MessageEditMediaModal";
    import MessageResendToModal from "@/MessageResendToModal";
    import PinnedMessagesModal from "@/PinnedMessagesModal";
    import PlayerModal from "@/PlayerModal";

    import queryMixin, {searchQueryParameter} from "@/queryMixin";
    import {copyCallLink, hasLength} from "@/utils";

    const reactOnAnswerThreshold = 3 * 1000; // ms
    const audio = new Audio("/call.mp3");
    const invitedVideoChatAlertTimeout = reactOnAnswerThreshold;

    let invitedVideoChatAlertTimer;

    export default {
        mixins: [queryMixin()],

        data () {
            return {
                drawer: this.$vuetify.breakpoint.lgAndUp,
                prevDrawer: false,
                invitedVideoChatId: 0,
                invitedVideoChatName: null,
                invitedVideoChatAlert: false,
                showWebsocketRestored: false,
                lastAnswered: 0,
                showSearchButton: true,
            }
        },
        components:{
            LoginModal,
            ChatEditModal,
            SimpleModal,
            ChooseAvatarModal,
            ChatParticipantsModal,
            PermissionsWarningModal,
            FindUserModal,
            FileUploadModal,
            FileListModal,
            VideoGlobalSettingsModal,
            FileTextEditModal,
            LanguageModal,
            VideoAddNewSourceModal,
            MessageEditModal,
            MessageEditLinkModal,
            MessageEditColorModal,
            NotificationsModal,
            MessageEditMediaModal,
            MessageResendToModal,
            PinnedMessagesModal,
            PlayerModal,
        },
        methods:{
            toggleLeftNavigation() {
                this.$data.drawer = !this.$data.drawer;
            },
            logout(){
                console.log("Logout");
                axios.post(`/api/logout`).then(({ data }) => {
                    this.$store.commit(UNSET_USER);
                    bus.$emit(LOGGED_OUT, null);
                });
            },
            goHome() {
                this.$router.push(({ name: chat_list_name}))
            },
            goBlog() {
                window.location.href = blog
            },
            goProfile() {
                this.$router.push(({ name: profile_self_name}))
            },
            onProfileClicked() {
                if (!this.isMobile()) {
                    this.goProfile();
                }
            },
            showCurrentUserSubtitle(){
                return hasLength(this?.currentUser.shortInfo)
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            },
            editChat() {
                bus.$emit(OPEN_CHAT_EDIT, this.chatId);
            },
            updateLastAnsweredTimestamp() {
                this.lastAnswered = +new Date();
            },
            onInfoClicked() {
                if (this.chatId) {
                    bus.$emit(OPEN_PARTICIPANTS_DIALOG, this.chatId);
                }
            },
            onNotificationsClicked() {
                bus.$emit(OPEN_NOTIFICATIONS_DIALOG);
            },
            addVideoSource() {
                bus.$emit(ADD_VIDEO_SOURCE_DIALOG);
            },
            addScreenSource() {
                bus.$emit(ADD_SCREEN_SOURCE);
            },
            onVideoCallInvited(data) {
                if ((+new Date() - this.lastAnswered) > reactOnAnswerThreshold) {
                    this.invitedVideoChatId = data.chatId;
                    this.invitedVideoChatName = data.chatName;
                    this.invitedVideoChatAlert = true;

                    // restart the timer
                    if (invitedVideoChatAlertTimer) {
                        clearInterval(invitedVideoChatAlertTimer);
                    }
                    // set auto-close snackbar
                    invitedVideoChatAlertTimer = setTimeout(()=>{
                        this.invitedVideoChatAlert = false;
                    }, invitedVideoChatAlertTimeout);

                    audio.play().catch(error => {
                        console.warn("Unable to play sound", error);
                        bus.$emit(OPEN_PERMISSIONS_WARNING_MODAL);
                    })
                }
            },
            onClickInvitation() {
                axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`).then(()=>{
                    const routerNewState = { name: videochat_name, params: { id: this.invitedVideoChatId }};
                    this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                    this.invitedVideoChatId = 0;
                    this.invitedVideoChatName = null;
                    this.invitedVideoChatAlert = false;
                    this.updateLastAnsweredTimestamp();
                });
            },
            onClickCancelInvitation() {
                axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`).then(()=>{
                    this.invitedVideoChatAlert = false;
                    this.updateLastAnsweredTimestamp();
                });
            },
            createCall() {
                console.debug("createCall");
                axios.put(`/api/video/${this.chatId}/dial/start`).then(()=>{
                    const routerNewState = { name: videochat_name};
                    this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                    this.updateLastAnsweredTimestamp();
                })
            },
            stopCall() {
                console.debug("stopping Call");
                const routerNewState = { name: chat_name, params: { leavingVideoAcceptableParam: true } };
                this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                this.updateLastAnsweredTimestamp();
            },
            copyCallLink() {
                copyCallLink(this.chatId)
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            findUser() {
                bus.$emit(OPEN_FIND_USER)
            },
            shouldDisplayFiles() {
                return this.chatId;
            },
            shouldPinnedMessages() {
                return this.chatId;
            },
            shouldDisplayEditChat() {
                return this.showChatEditButton;
            },
            shouldDisplayLogout() {
                return this.currentUser != null;
            },
            shouldDisplayProfile() {
                return this.currentUser != null;
            },
            shouldDisplayVideoSettings() {
                return this.chatId;
            },
            onPressWebsocketRestored() {
                this.showWebsocketRestored = false;
                bus.$emit(REFRESH_ON_WEBSOCKET_RESTORED);
            },
            onWsRestored() {
                this.showWebsocketRestored = true;
            },
            onVideoOpened() {
                this.prevDrawer = this.drawer;
                this.drawer = false;
            },
            onVideoClosed() {
                this.drawer = this.prevDrawer;
            },
            displayChatFiles() {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId});
            },
            openPinnedMessages() {
                bus.$emit(OPEN_PINNED_MESSAGES_MODAL);
            },
            openVideoSettings() {
                bus.$emit(OPEN_VIDEO_SETTINGS);
            },
            openLocale() {
                bus.$emit(OPEN_LANGUAGE_MODAL);
            },
            startRecord() {
                axios.put(`/api/video/${this.chatId}/record/start`);
                this.$store.commit(SET_INITIALIZING_STARTING_VIDEO_RECORD, true)
            },
            stopRecord() {
                axios.put(`/api/video/${this.chatId}/record/stop`);
                this.$store.commit(SET_INITIALIZING_STOPPING_VIDEO_RECORD, true)
            },
            onVideRecordingChanged(e) {
                if (this.isVideoRoute()) {
                    this.$store.commit(SET_SHOW_RECORD_START_BUTTON, !e.recordInProgress);
                    this.$store.commit(SET_SHOW_RECORD_STOP_BUTTON, e.recordInProgress);
                } else if (e.recordInProcess) {
                    this.$store.commit(SET_SHOW_RECORD_START_BUTTON, !e.recordInProgress);
                    this.$store.commit(SET_SHOW_RECORD_STOP_BUTTON, e.recordInProgress);
                }
                if (this.initializingStaringVideoRecord && e.recordInProgress) {
                    this.$store.commit(SET_INITIALIZING_STARTING_VIDEO_RECORD, false)
                }
                if (this.initializingStoppingVideoRecord && !e.recordInProgress) {
                    this.$store.commit(SET_INITIALIZING_STOPPING_VIDEO_RECORD, false)
                }
            },
            resetInput() {
                this.searchString = null;
                this.showSearchButton = true;
            },
            // reacts on input into search field
            searchStringChanged(searchString) {
                console.debug("doSearch in App", searchString);

                const routerNewState = {name: this.$route.name};
                if (hasLength(searchString)) {
                    routerNewState.query = {[searchQueryParameter]: searchString};
                } else {
                    this.showSearchButton = true;
                }
                // in order not to reset on initialization
                const fullHash = this.getRouteHash(true);
                if (hasLength(fullHash)) {
                    routerNewState.hash = fullHash;
                }
                this.$router.push(routerNewState).catch(()=>{});
            },
            onProfileSet(){
                this.$store.dispatch(FETCH_NOTIFICATIONS);
            },
            onLoggedOut() {
                this.resetVariables();
            },
            resetVariables() {
                this.$store.commit(UNSET_NOTIFICATIONS);
            },
            onOpenSearch() {
                this.showSearchButton = false;
            },
            onMicrophone() {
                bus.$emit(SET_LOCAL_MICROPHONE_MUTED, false);
            },
            offMicrophone() {
                bus.$emit(SET_LOCAL_MICROPHONE_MUTED, true);
            }
        },
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
                showCallButton: GET_SHOW_CALL_BUTTON,
                showHangButton: GET_SHOW_HANG_BUTTON,
                showRecordStartButton: GET_SHOW_RECORD_START_BUTTON,
                showRecordStopButton: GET_SHOW_RECORD_STOP_BUTTON,
                videoChatUsersCount: GET_VIDEO_CHAT_USERS_COUNT,
                showChatEditButton: GET_SHOW_CHAT_EDIT_BUTTON,
                chatId: GET_CHAT_ID,
                title: GET_TITLE,
                chatAvatar: GET_AVATAR,
                chatUsersCount: GET_CHAT_USERS_COUNT,
                isShowSearch: GET_SHOW_SEARCH,
                searchName: GET_SEARCH_NAME,
                showAlert: GET_SHOW_ALERT,
                lastError: GET_LAST_ERROR,
                errorColor: GET_ERROR_COLOR,
                shouldPhoneBlink: GET_SHOULD_PHONE_BLINK,
                tetATet: GET_TET_A_TET,
                showMicrophoneOnButton: GET_SHOW_MICROPHONE_ON_BUTTON,
                showMicrophoneOffButton: GET_SHOW_MICROPHONE_OFF_BUTTON,
                canShowMicrophoneButton: GET_CAN_SHOW_MICROPHONE_BUTTON,
                initializingStaringVideoRecord: GET_INITIALIZING_STARTING_VIDEO_RECORD,
                initializingStoppingVideoRecord: GET_INITIALIZING_STOPPING_VIDEO_RECORD,
            }), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                return this.currentUser.avatar;
            },
            notificationsCount() {
                return this.$store.getters[GET_NOTIFICATIONS].length
            },
            hasSearchString() {
                return hasLength(this.searchString)
            }
        },
        mounted() {
        },
        created() {
            bus.$on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            bus.$on(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
            bus.$on(PROFILE_SET, this.onProfileSet);
            bus.$on(LOGGED_OUT, this.onLoggedOut);
            bus.$on(WEBSOCKET_RESTORED, this.onWsRestored);
            bus.$on(VIDEO_OPENED, this.onVideoOpened);
            bus.$on(VIDEO_CLOSED, this.onVideoClosed);

            this.$store.dispatch(FETCH_AVAILABLE_OAUTH2_PROVIDERS).then(() => {
                this.$store.dispatch(FETCH_USER_PROFILE);
            })

            this.initQueryAndWatcher();
        },
        destroyed() {
            this.closeQueryWatcher();

            bus.$off(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            bus.$off(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
            bus.$off(PROFILE_SET, this.onProfileSet);
            bus.$off(LOGGED_OUT, this.onLoggedOut);
            bus.$off(WEBSOCKET_RESTORED, this.onWsRestored);
            bus.$off(VIDEO_OPENED, this.onVideoOpened);
            bus.$off(VIDEO_CLOSED, this.onVideoClosed);
        },
    }
</script>

<style lang="stylus">
    html {
        overflow-y auto
    }

    .row {
      margin-top: 0px !important;
      margin-bottom: 0px !important;
    }

</style>

<style scoped lang="stylus">
    .call-blink {
        animation: blink 0.5s infinite;
    }

    @keyframes blink {
        50% { opacity: 30% }
    }

    .app-title {
        &-text {
            font-size: .875rem;
            font-weight: 500;
            letter-spacing: .0892857143em;
            text-indent: .0892857143em;
        }

        &-subtext {
            font-size: .7rem;
            letter-spacing: initial;
            text-transform: initial;
            opacity: 50%
        }

        &-hoverable {
            color white
        }

        &-hoverable:hover {
            background-color: #4e5fbb;
            border-radius: 4px;
        }
    }

    .chat-avatar {
        display: block;
        max-width: 36px;
        max-height: 36px;
        width: auto;
        height: auto;
    }

</style>
