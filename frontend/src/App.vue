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
                <v-list-item two-line v-if="currentUser">
                    <v-list-item-avatar  v-if="currentUser.avatar">
                        <img :src="currentUserAvatar"/>
                    </v-list-item-avatar>

                    <v-list-item-content>
                        <v-list-item-title class="user-login">{{currentUser.login}}</v-list-item-title>
                    </v-list-item-content>
                </v-list-item>
            </template>

            <v-divider></v-divider>

            <v-list dense>
                <v-list-item @click.prevent="goHome()" :href="require('./routes').root">
                    <v-list-item-icon><v-icon>mdi-forum</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.chats') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="createChat()">
                    <v-list-item-icon><v-icon>mdi-plus</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title id="new-chat-dialog-button">{{ $vuetify.lang.t('$vuetify.new_chat') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="displayChatFiles()" v-if="shouldDisplayFiles()">
                    <v-list-item-icon><v-icon>mdi-file-download</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.files') }}</v-list-item-title></v-list-item-content>
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
            <v-btn v-if="showHangButton && !isMobile()" icon @click="addScreenSource()" :title="$vuetify.lang.t('$vuetify.screen_share')"><v-icon>mdi-monitor-screenshot</v-icon></v-btn>
            <v-btn v-if="showHangButton" icon @click="addVideoSource()" :title="$vuetify.lang.t('$vuetify.source_add')"><v-icon>mdi-video-plus</v-icon></v-btn>
            <v-btn v-if="showRecordStartButton" icon @click="startRecord()" :loading="initializingStaringVideoRecord" :title="$vuetify.lang.t('$vuetify.start_record')">
                <v-icon>mdi-record-rec</v-icon>
            </v-btn>
            <v-btn v-if="showRecordStopButton" icon @click="stopRecord()" :loading="initializingStoppingVideoRecord" :title="$vuetify.lang.t('$vuetify.stop_record')">
                <v-icon color="red">mdi-record-rec</v-icon>
            </v-btn>
            <v-badge
                v-if="showCallButton || showHangButton"
                :content="videoChatUsersCount"
                :value="videoChatUsersCount"
                color="green"
                overlap
                offset-y="1.8em"
            >
                <v-btn v-if="showCallButton" icon @click="createCall()" :title="$vuetify.lang.t('$vuetify.create_call')">
                    <v-icon color="green">mdi-phone</v-icon>
                </v-btn>
                <v-btn v-if="showHangButton" icon @click="stopCall()" :title="$vuetify.lang.t('$vuetify.leave_call')">
                    <v-icon color="red">mdi-phone</v-icon>
                </v-btn>
            </v-badge>

            <v-spacer></v-spacer>
            <v-toolbar-title color="white" class="d-flex flex-column px-2 app-title" :class="chatId ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked" :style="{'cursor': chatId ? 'pointer' : 'default'}">
                <div class="align-self-center app-title-text">{{title}}</div>
                <div v-if="chatUsersCount" class="align-self-center app-title-subtext">
                    {{ chatUsersCount }} {{ $vuetify.lang.t('$vuetify.participants') }}</div>
            </v-toolbar-title>
            <v-spacer></v-spacer>

            <v-card light v-if="isShowSearch">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line @input="clearHash()" v-model="searchString" :label="searchName" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
            </v-card>

            <v-badge
                :content="notificationsCount"
                :value="notificationsCount"
                color="red"
                overlap
                offset-y="1.8em"
            >
                <v-btn
                    icon :title="$vuetify.lang.t('$vuetify.notifications')"
                    @click="onNotificationsClicked()"
                >
                    <v-icon>mdi-bell</v-icon>
                </v-btn>
            </v-badge>
        </v-app-bar>


        <v-main>
            <v-container fluid class="ma-0 pa-0">
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
                <ChatEdit/>
                <ChatParticipants/>
                <SimpleModal/>
                <PermissionsWarning/>
                <ChooseAvatar/>
                <FindUser/>
                <FileUploadModal/>
                <FileListModal/>
                <VideoGlobalSettings/>
                <FileTextEditModal/>
                <LanguageModal/>
                <VideoAddNewSource/>
                <MessageEditModal v-if="isMobile()"/>
                <MessageEditLinkModal/>
                <MessageEditColorModal/>
                <NotificationsModal/>

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
        GET_USER,
        GET_VIDEO_CHAT_USERS_COUNT,
        UNSET_USER,
        GET_SHOW_RECORD_START_BUTTON,
        GET_SHOW_RECORD_STOP_BUTTON,
        SET_SHOW_RECORD_START_BUTTON,
        SET_SHOW_RECORD_STOP_BUTTON,
        FETCH_NOTIFICATIONS,
        GET_NOTIFICATIONS, UNSET_NOTIFICATIONS, FETCH_AVAILABLE_OAUTH2_PROVIDERS, GET_SEARCH_NAME, SET_TITLE
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
        OPEN_SIMPLE_MODAL,
        CLOSE_SIMPLE_MODAL,
        VIDEO_RECORDING_CHANGED,
        OPEN_NOTIFICATIONS_DIALOG,
        PROFILE_SET,
    } from "./bus";
    import ChatEdit from "./ChatEdit";
    import {chat_name, profile_self_name, chat_list_name, videochat_name} from "./routes";
    import SimpleModal from "./SimpleModal";
    import ChooseAvatar from "./ChooseAvatar";
    import ChatParticipants from "./ChatParticipants";
    import PermissionsWarning from "./PermissionsWarning";
    import FindUser from "./FindUser";
    import FileUploadModal from './FileUploadModal';
    import FileListModal from "./FileListModal";
    import VideoGlobalSettings from './VideoGlobalSettings';
    import FileTextEditModal from "./FileTextEditModal";
    import LanguageModal from "./LanguageModal";
    import VideoAddNewSource from "@/VideoAddNewSource";
    import MessageEditModal from "@/MessageEditModal";
    import MessageEditLinkModal from "@/MessageEditLinkModal";
    import MessageEditColorModal from "@/MessageEditColorModal";
    import NotificationsModal from "@/NotificationsModal";

    import queryMixin, {searchQueryParameter} from "@/queryMixin";
    import {hasLength} from "@/utils";

    const reactOnAnswerThreshold = 3 * 1000; // ms
    const audio = new Audio("/call.mp3");
    const invitedVideoChatAlertTimeout = reactOnAnswerThreshold;

    let invitedVideoChatAlertTimer;

    export default {
        mixins: [queryMixin()],

        data () {
            return {
                drawer: this.$vuetify.breakpoint.lgAndUp,
                invitedVideoChatId: 0,
                invitedVideoChatName: null,
                invitedVideoChatAlert: false,
                showWebsocketRestored: false,
                lastAnswered: 0,
                initializingStaringVideoRecord: false,
                initializingStoppingVideoRecord: false,
            }
        },
        components:{
            LoginModal,
            ChatEdit,
            SimpleModal,
            ChooseAvatar,
            ChatParticipants,
            PermissionsWarning,
            FindUser,
            FileUploadModal,
            FileListModal,
            VideoGlobalSettings,
            FileTextEditModal,
            LanguageModal,
            VideoAddNewSource,
            MessageEditModal,
            MessageEditLinkModal,
            MessageEditColorModal,
            NotificationsModal,
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
            goProfile() {
                this.$router.push(({ name: profile_self_name}))
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
            createCall() {
                console.debug("createCall");
                const routerNewState = { name: videochat_name};
                this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                this.updateLastAnsweredTimestamp();
            },
            stopCall() {
                console.debug("stopping Call");
                const routerNewState = { name: chat_name, params: { leavingVideoAcceptableParam: true } };
                this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                this.updateLastAnsweredTimestamp();
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
                const routerNewState = { name: videochat_name, params: { id: this.invitedVideoChatId }};
                this.navigateToWithPreservingSearchStringInQuery(routerNewState);
                axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`);
                this.invitedVideoChatId = 0;
                this.invitedVideoChatName = null;
                this.invitedVideoChatAlert = false;
                this.updateLastAnsweredTimestamp();
            },
            onClickCancelInvitation() {
                this.invitedVideoChatAlert = false;
                axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`);
                this.updateLastAnsweredTimestamp();
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
            displayChatFiles() {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId});
            },
            openVideoSettings() {
                bus.$emit(OPEN_VIDEO_SETTINGS);
            },
            openLocale() {
                bus.$emit(OPEN_LANGUAGE_MODAL);
            },
            startRecord() {
                axios.put(`/api/video/${this.chatId}/record/start`);
                this.initializingStaringVideoRecord = true;
            },
            stopRecord() {
                axios.put(`/api/video/${this.chatId}/record/stop`);
                this.initializingStoppingVideoRecord = true;
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
                    this.initializingStaringVideoRecord = false;
                }
                if (this.initializingStoppingVideoRecord && !e.recordInProgress) {
                    this.initializingStoppingVideoRecord = false;
                }
            },
            resetInput() {
                this.searchString = null;
            },
            searchStringChanged(searchString) {
                console.debug("doSearch", searchString);

                const currentRouteName = this.$route.name;
                const routerNewState = {name: currentRouteName};
                if (searchString && searchString != "") {
                    routerNewState.query = {[searchQueryParameter]: searchString};
                }
                if (hasLength(this.getHash(true))) {
                    routerNewState.hash = this.getHash(true);
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
                chatUsersCount: GET_CHAT_USERS_COUNT,
                isShowSearch: GET_SHOW_SEARCH,
                searchName: GET_SEARCH_NAME,
                showAlert: GET_SHOW_ALERT,
                lastError: GET_LAST_ERROR,
                errorColor: GET_ERROR_COLOR,
            }), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                return this.currentUser.avatar;
            },
            notificationsCount() {
                return this.$store.getters[GET_NOTIFICATIONS].length
            }
        },
        mounted() {
        },
        created() {
            bus.$on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            bus.$on(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
            bus.$on(PROFILE_SET, this.onProfileSet);
            bus.$on(LOGGED_OUT, this.onLoggedOut);

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
        color #7481c9
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

</style>