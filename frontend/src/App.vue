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
                    <v-list-item-avatar>
                        <img :src="currentUserAvatar"/>
                    </v-list-item-avatar>

                    <v-list-item-content>
                        <v-list-item-title>{{currentUser.login}}</v-list-item-title>
                        <v-list-item-subtitle>Logged In</v-list-item-subtitle>
                    </v-list-item-content>
                </v-list-item>
            </template>

            <v-divider></v-divider>

            <v-list dense>
                <v-list-item
                        v-for="item in getAppBarItems()"
                        :key="item.title"
                        @click="item.clickFunction"
                >
                    <v-list-item-icon>
                        <v-icon>{{ item.icon }}</v-icon>
                    </v-list-item-icon>

                    <v-list-item-content>
                        <v-list-item-title>{{ item.title }}</v-list-item-title>
                    </v-list-item-content>
                </v-list-item>
            </v-list>
        </v-navigation-drawer>


        <v-app-bar
                color="indigo"
                dark
                app
                :clipped-left="true"
        >
            <v-app-bar-nav-icon @click="toggleLeftNavigation"></v-app-bar-nav-icon>
            <v-btn v-if="showHangButton && !shareScreen && $vuetify.breakpoint.smAndUp" icon @click="shareScreenStart()"><v-icon>mdi-monitor-screenshot</v-icon></v-btn>
            <v-btn v-if="showHangButton && shareScreen" icon @click="shareScreenStop()"><v-icon>mdi-stop</v-icon></v-btn>
            <v-btn v-if="showChatEditButton" icon @click="editChat">
                <v-icon>mdi-lead-pencil</v-icon>
            </v-btn>
            <v-badge
                v-if="showCallButton || showHangButton"
                :content="usersCount"
                :value="usersCount"
                color="green"
                overlap
                offset-y="1.8em"
            >
                <v-btn v-if="showCallButton" icon @click="createCall">
                    <v-icon color="green">mdi-phone</v-icon>
                </v-btn>
                <v-btn v-if="showHangButton" icon @click="stopCall">
                    <v-icon color="red">mdi-phone</v-icon>
            </v-btn>
            </v-badge>

            <v-spacer></v-spacer>
            <v-btn class="ma-2" text color="white" @click="onInfoClicked" :disabled="!chatId">
                <div class="d-flex flex-column">
                    <div style="text-transform: initial">{{title}}</div>
                    <div v-if="chatUsersCount" style="font-size: 0.8em !important; letter-spacing: initial; text-transform: initial; opacity: 50%">
                        {{ chatUsersCount }} participants</div>
                </div>
            </v-btn>
            <v-alert
                v-model="invitedVideoChatAlert"
                close-text="Close Alert"
                dismissible
                prominent
                class="mb-0 mt-0 ml-0 mr-1 pb-0 pt-0 px-4"
                color="success"
            >
                <v-row align="center" class="call-blink" :key="callReblinkCounter">
                    <v-col class="grow" v-if="$vuetify.breakpoint.smAndUp">
                        You are called
                    </v-col>
                    <v-col class="shrink ma-0 pa-0">
                        <v-btn icon @click="onClickInvitation"><v-icon color="white">mdi-phone</v-icon></v-btn>
                    </v-col>
                </v-row>
            </v-alert>

            <v-spacer></v-spacer>
            <v-tooltip bottom v-if="!wsConnected">
                <template v-slot:activator="{ on, attrs }">
                    <v-icon color="error" class="mr-2" v-bind="attrs" v-on="on">mdi-lan-disconnect</v-icon>
                </template>
                <span>Websocket is not connected</span>
            </v-tooltip>
            <v-card light v-if="showSearch">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line v-model="searchChatString" clearable clear-icon="mdi-close-circle"></v-text-field>
            </v-card>
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

                <LoginModal/>
                <ChatEdit/>
                <ChatParticipants/>
                <SimpleModal/>
                <PermissionsWarning/>
                <ChooseAvatar/>
                <router-view/>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import 'typeface-roboto'
    import axios from 'axios';
    import LoginModal from "./LoginModal";
    import {mapGetters} from 'vuex'
    import {CHANGE_SEARCH_STRING, FETCH_USER_PROFILE, GET_USER, UNSET_USER} from "./store";
    import bus, {
        AUDIO_MUTED,
        AUDIO_START_MUTING,
        CHANGE_PHONE_BUTTON,
        CHANGE_TITLE,
        CHANGE_WEBSOCKET_STATUS,
        LOGGED_OUT,
        OPEN_CHAT_EDIT,
        OPEN_INFO_DIALOG,
        OPEN_PERMISSIONS_WARNING_MODAL,
        SHARE_SCREEN_START, SHARE_SCREEN_STATE_CHANGED, SHARE_SCREEN_STOP,
        VIDEO_CALL_CHANGED,
        VIDEO_CALL_INVITED, VIDEO_COMPONENT_DESTROYED, VIDEO_MUTED, VIDEO_START_MUTING,
    } from "./bus";
    import ChatEdit from "./ChatEdit";
    import debounce from "lodash/debounce";
    import {chat_name, profile_name, chat_list_name, videochat_name} from "./routes";
    import SimpleModal from "./SimpleModal";
    import ChooseAvatar from "./ChooseAvatar";
    import {getCorrectUserAvatar} from "./utils";
    import ChatParticipants from "./ChatParticipants";
    import PermissionsWarning from "./PermissionsWarning";

    const audio = new Audio("/call.mp3");

    export default {
        data () {
            return {
                title: "",
                appBarItems: [
                    { title: 'Unmute audio', icon: 'mdi-microphone', clickFunction: this.toggleMuteAudio, requireAuthenticated: true, displayCondition: this.shouldDisplayAudioUnmute},
                    { title: 'Mute audio', icon: 'mdi-microphone-off', clickFunction: this.toggleMuteAudio, requireAuthenticated: true, displayCondition: this.shouldDisplayAudioMute},
                    { title: 'Unmute video', icon: 'mdi-video', clickFunction: this.toggleMuteVideo, requireAuthenticated: true, displayCondition: this.shouldDisplayVideoUnmute},
                    { title: 'Mute video', icon: 'mdi-video-off', clickFunction: this.toggleMuteVideo, requireAuthenticated: true, displayCondition: this.shouldDisplayVideoMute},

                    { title: 'New chat', icon: 'mdi-plus-circle-outline', clickFunction: this.createChat, requireAuthenticated: true},
                    { title: 'Chats', icon: 'mdi-home-city', clickFunction: this.goHome, requireAuthenticated: false },
                    { title: 'My Account', icon: 'mdi-account', clickFunction: this.goProfile, requireAuthenticated: true },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout, requireAuthenticated: true },
                ],
                drawer: this.$vuetify.breakpoint.lgAndUp,
                lastError: "",
                showAlert: false,
                searchChatString: "",
                showSearch: false,
                showChatEditButton: false,
                showCallButton: false,
                showHangButton: false,
                chatId: null,
                chatUsersCount: null,
                chatEditId: null, // nullable if non-chat admin
                wsConnected: false,
                usersCount: 0,
                invitedVideoChatId: 0,
                invitedVideoChatAlert: false,
                callReblinkCounter: 0,
                shareScreen: false,
                audioMuted: false,
                videoMuted: false
            }
        },
        components:{
            LoginModal,
            ChatEdit,
            SimpleModal,
            ChooseAvatar,
            ChatParticipants,
            PermissionsWarning
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
                this.$router.push(({ name: profile_name}))
            },
            onError(errText){
                this.showAlert = true;
                this.lastError = errText;
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            },
            editChat() {
                bus.$emit(OPEN_CHAT_EDIT, this.chatEditId);
            },
            doSearch(searchString) {
                this.$store.dispatch(CHANGE_SEARCH_STRING, searchString);
            },
            getAppBarItems(){
                return this.appBarItems.filter((value, index) => {
                    if (value.requireAuthenticated) {
                        return this.currentUser
                    } else {
                        return true
                    }
                }).filter((value, index) => {
                    if (value.displayCondition) {
                        return value.displayCondition();
                    } else {
                        return true
                    }
                })
            },
            changeTitle({title, isShowSearch, isShowChatEditButton, chatEditId, chatId, chatUsersCount}) {
                this.title = title;
                this.showSearch = isShowSearch;
                this.showChatEditButton = isShowChatEditButton;
                this.chatEditId = chatEditId;
                this.chatId = chatId;
                this.chatUsersCount = chatUsersCount;
            },
            changePhoneButton({show, call}) {
                console.log("changePhoneButton", show, call);
                if (!show) {
                    this.showCallButton = false;
                    this.showHangButton = false;
                } else {
                    if (call) {
                        this.showCallButton = true;
                        this.showHangButton = false;
                    } else {
                        this.showCallButton = false;
                        this.showHangButton = true;
                    }
                }
            },
            onVideoDestroyed() {
                this.chatUsersCount = 0;
                this.shareScreen = false;
                this.audioMuted = false;
                this.videoMuted = false;
            },
            createCall() {
                console.log("createCall");
                this.$router.push({ name: videochat_name});
            },
            stopCall() {
                console.log("stopCall");
                this.$router.push({ name: chat_name});
            },
            shareScreenStart() {
                bus.$emit(SHARE_SCREEN_START);
            },
            shareScreenStop() {
                bus.$emit(SHARE_SCREEN_STOP);
            },
            onChangeWsStatus(value) {
                this.wsConnected = value;
            },
            onInfoClicked() {
                bus.$emit(OPEN_INFO_DIALOG, this.chatId);
            },
            onVideoCallChanged(data) {
                this.usersCount = data.usersCount
            },
            onVideoCallInvited(data) {
                this.invitedVideoChatId = data.chatId;
                this.invitedVideoChatAlert = true;
                ++this.callReblinkCounter;
                audio.play().catch(error => {
                    console.warn("Unable to play sound", error);
                  bus.$emit(OPEN_PERMISSIONS_WARNING_MODAL);
                })
            },
            onClickInvitation() {
                this.$router.push({ name: videochat_name, params: { id: this.invitedVideoChatId }});
                this.invitedVideoChatAlert = false;
            },
            onShareScreenStateChanged(newState) {
                this.shareScreen = newState;
            },
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            toggleMuteAudio() {
                bus.$emit(AUDIO_START_MUTING, !this.audioMuted)
            },
            toggleMuteVideo() {
                bus.$emit(VIDEO_START_MUTING, !this.videoMuted)
            },
            onAudioMuteChanged(newState) {
                this.audioMuted = newState;
            },
            onVideoMuteChanged(newState) {
                this.videoMuted = newState;
            },
            shouldDisplayAudioUnmute() {
                return this.isVideoRoute() && this.audioMuted;
            },
            shouldDisplayAudioMute() {
                return this.isVideoRoute() && !this.audioMuted;
            },
            shouldDisplayVideoUnmute() {
                return this.isVideoRoute() && this.videoMuted;
            },
            shouldDisplayVideoMute() {
                return this.isVideoRoute() && !this.videoMuted;
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                return getCorrectUserAvatar(this.currentUser.avatar);
            },
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(CHANGE_TITLE, this.changeTitle);
            bus.$on(CHANGE_PHONE_BUTTON, this.changePhoneButton);
            bus.$on(CHANGE_WEBSOCKET_STATUS, this.onChangeWsStatus);
            bus.$on(VIDEO_CALL_CHANGED, this.onVideoCallChanged);
            bus.$on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            bus.$on(SHARE_SCREEN_STATE_CHANGED, this.onShareScreenStateChanged);
            bus.$on(AUDIO_MUTED, this.onAudioMuteChanged);
            bus.$on(VIDEO_MUTED, this.onVideoMuteChanged);
            bus.$on(VIDEO_COMPONENT_DESTROYED, this.onVideoDestroyed);
        },
        watch: {
            searchChatString (searchString) {
                this.doSearch(searchString);
            },
        },

    }
</script>

<style scoped lang="stylus">
    .call-blink {
        animation: blink 0.5s;
        animation-iteration-count: 5;
    }

    @keyframes blink {
        50% { opacity: 10% }
    }
</style>