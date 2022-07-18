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
                    <v-list-item-icon><v-icon>mdi-home-city</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.chats') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="displayChatFiles()" v-if="shouldDisplayFiles()">
                    <v-list-item-icon><v-icon>mdi-file-download</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.files') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="findUser()">
                    <v-list-item-icon><v-icon>mdi-magnify</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title>{{ $vuetify.lang.t('$vuetify.find_user') }}</v-list-item-title></v-list-item-content>
                </v-list-item>

                <v-list-item @click="createChat()">
                    <v-list-item-icon><v-icon>mdi-plus-circle-outline</v-icon></v-list-item-icon>
                    <v-list-item-content><v-list-item-title id="new-chat-dialog-button">{{ $vuetify.lang.t('$vuetify.new_chat') }}</v-list-item-title></v-list-item-content>
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
                :color="wsConnected ? 'indigo' : 'error'"
                dark
                app
                id="myAppBar"
                :clipped-left="true"
                dense
        >
            <v-app-bar-nav-icon @click="toggleLeftNavigation"></v-app-bar-nav-icon>
            <v-btn v-if="showHangButton && $vuetify.breakpoint.smAndUp" icon @click="addScreenSource()"><v-icon>mdi-monitor-screenshot</v-icon></v-btn>
            <v-btn v-if="showHangButton" icon @click="addVideoSource()"><v-icon>mdi-video-plus</v-icon></v-btn>
            <v-badge
                v-if="showCallButton || showHangButton"
                :content="videoChatUsersCount"
                :value="videoChatUsersCount"
                color="green"
                overlap
                offset-y="1.8em"
            >
                <v-btn v-if="showCallButton" icon @click="createCall()">
                    <v-icon color="green">mdi-phone</v-icon>
                </v-btn>
                <v-btn v-if="showHangButton" icon @click="stopCall()">
                    <v-icon color="red">mdi-phone</v-icon>
                </v-btn>
            </v-badge>

            <v-spacer></v-spacer>
            <v-btn class="ma-2" text color="white" @click="onInfoClicked" :disabled="!chatId">
                <div class="d-flex flex-column">
                    <div style="text-transform: initial">{{title}}</div>
                    <div v-if="chatUsersCount" style="font-size: 0.8em !important; letter-spacing: initial; text-transform: initial; opacity: 50%">
                        {{ chatUsersCount }} {{ $vuetify.lang.t('$vuetify.participants') }}</div>
                </div>
            </v-btn>
            <v-spacer></v-spacer>

            <v-card light v-if="isShowSearch">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" clearable clear-icon="mdi-close-circle"></v-text-field>
            </v-card>

            <v-tooltip bottom v-if="!wsConnected">
                <template v-slot:activator="{ on, attrs }">
                    <v-icon color="white" class="ml-2" v-bind="attrs" v-on="on">mdi-lan-disconnect</v-icon>
                </template>
                <span>{{ $vuetify.lang.t('$vuetify.websocket_not_connected') }}</span>
            </v-tooltip>
        </v-app-bar>


        <v-main>
            <v-container fluid class="ma-0 pa-0">
                <v-snackbar v-model="showAlert" color="error" timeout="-1" :multi-line="true">
                    {{ lastError }}

                    <template v-slot:action="{ attrs }">
                        <v-btn
                            text
                            v-bind="attrs"
                            @click="showAlert = false"
                        >
                            Close
                        </v-btn>
                    </template>
                </v-snackbar>
                <v-snackbar v-model="showWebsocketRestored" color="black" timeout="-1" :multi-line="true">
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
                <v-snackbar v-model="invitedVideoChatAlert" class="call-blink" color="success" timeout="-1" :multi-line="true" :key="callReblinkCounter" top>
                    {{ $vuetify.lang.t('$vuetify.you_called', invitedVideoChatId, invitedVideoChatName) }}
                    <template v-slot:action="{ attrs }">
                        <v-btn icon v-bind="attrs" @click="onClickInvitation()"><v-icon color="white">mdi-phone</v-icon></v-btn>
                        <v-btn icon v-bind="attrs" @click="invitedVideoChatAlert = false"><v-icon color="white">mdi-close-circle</v-icon></v-btn>
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
                <MessageEditModal/>

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
        FETCH_USER_PROFILE, GET_CHAT_ID, GET_CHAT_USERS_COUNT,
        GET_SHOW_CALL_BUTTON,
        GET_SHOW_CHAT_EDIT_BUTTON,
        GET_SHOW_HANG_BUTTON, GET_SHOW_SEARCH,
        GET_TITLE,
        GET_USER,
        GET_VIDEO_CHAT_USERS_COUNT,
        UNSET_USER
    } from "./store";
    import bus, {
        CHANGE_WEBSOCKET_STATUS,
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
        ADD_SCREEN_SOURCE, SEARCH_STRING_CHANGED,
    } from "./bus";
    import ChatEdit from "./ChatEdit";
    import {chat_name, profile_self_name, chat_list_name, videochat_name} from "./routes";
    import SimpleModal from "./SimpleModal";
    import ChooseAvatar from "./ChooseAvatar";
    import {setIcon} from "./utils";
    import ChatParticipants from "./ChatParticipants";
    import PermissionsWarning from "./PermissionsWarning";
    import FindUser from "./FindUser";
    import FileUploadModal from './FileUploadModal';
    import FileListModal from "./FileListModal";
    import VideoGlobalSettings from './VideoGlobalSettings';
    import FileTextEditModal from "./FileTextEditModal";
    import LanguageModal from "./LanguageModal";
    import {getData} from "@/centrifugeConnection";
    import VideoAddNewSource from "@/VideoAddNewSource";
    import debounce from "lodash/debounce";
    import MessageEditModal from "@/MessageEditModal";

    const audio = new Audio("/call.mp3");

    export default {
        data () {
            return {
                drawer: this.$vuetify.breakpoint.lgAndUp,
                lastError: "",
                showAlert: false,
                wsConnected: false,
                invitedVideoChatId: 0,
                invitedVideoChatName: null,
                invitedVideoChatAlert: false,
                callReblinkCounter: 0,
                showWebsocketRestored: false,
                searchString: "",
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
            onError(errText){
                this.showAlert = true;
                this.lastError = errText;
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            },
            editChat() {
                bus.$emit(OPEN_CHAT_EDIT, this.chatId);
            },
            createCall() {
                console.log("createCall");
                this.$router.push({ name: videochat_name});
            },
            stopCall() {
                console.log("stopping Call");
                this.$router.push({ name: chat_name});
            },
            onChangeWsStatus({connected, wasInitialized}) {
                console.log("onChangeWsStatus: connected", connected, "wasInitialized", wasInitialized)
                this.wsConnected = connected;
                if (connected && wasInitialized) {
                    this.showWebsocketRestored = true;
                }
                if (connected) {
                    this.centrifuge.namedRPC("check_for_new_messages").then(value => {
                        console.debug("New messages response", value);
                        if (getData(value)) {
                            const currentNewMessages = getData(value).allUnreadMessages > 0;
                            setIcon(currentNewMessages)
                        }
                    })
                }
            },
            onInfoClicked() {
                bus.$emit(OPEN_PARTICIPANTS_DIALOG, this.chatId);
            },
            addVideoSource() {
                bus.$emit(ADD_VIDEO_SOURCE_DIALOG);
            },
            addScreenSource() {
                bus.$emit(ADD_SCREEN_SOURCE);
            },
            onVideoCallInvited(data) {
                this.invitedVideoChatId = data.chatId;
                this.invitedVideoChatName = data.chatName;
                this.invitedVideoChatAlert = true;
                ++this.callReblinkCounter;
                audio.play().catch(error => {
                    console.warn("Unable to play sound", error);
                  bus.$emit(OPEN_PERMISSIONS_WARNING_MODAL);
                })
            },
            onClickInvitation() {
                this.$router.push({ name: videochat_name, params: { id: this.invitedVideoChatId }});
                this.invitedVideoChatId = 0;
                this.invitedVideoChatName = null;
                this.invitedVideoChatAlert = false;
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
            doSearch(searchString) {
                if (!searchString || searchString == "") {
                    bus.$emit(SEARCH_STRING_CHANGED, null);
                    return;
                }
                searchString = searchString.trim();
                bus.$emit(SEARCH_STRING_CHANGED, searchString);
            }
        },
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
                showCallButton: GET_SHOW_CALL_BUTTON,
                showHangButton: GET_SHOW_HANG_BUTTON,
                videoChatUsersCount: GET_VIDEO_CHAT_USERS_COUNT,
                showChatEditButton: GET_SHOW_CHAT_EDIT_BUTTON,
                chatId: GET_CHAT_ID,
                title: GET_TITLE,
                chatUsersCount: GET_CHAT_USERS_COUNT,
                isShowSearch: GET_SHOW_SEARCH,
            }), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                return this.currentUser.avatar;
            },
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            bus.$on(CHANGE_WEBSOCKET_STATUS, this.onChangeWsStatus);
            bus.$on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            this.doSearch = debounce(this.doSearch, 700, {leading:false, trailing:true});
        },
        watch: {
            searchString (searchString) {
                this.doSearch(searchString);
            },
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
        animation: blink 0.5s;
        animation-iteration-count: 5;
    }

    @keyframes blink {
        50% { opacity: 10% }
    }
</style>