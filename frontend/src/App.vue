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
                        <v-list-item-title>{{currentUser.login}}</v-list-item-title>
                        <v-list-item-subtitle>Logged In</v-list-item-subtitle>
                    </v-list-item-content>
                </v-list-item>
            </template>

            <v-divider></v-divider>

            <v-list dense>
                <v-list-item v-if="isVideoRoute()">
                    <v-menu
                        bottom
                        offset-y
                        absolute
                    >
                        <template v-slot:activator="{ on, attrs }">
                            <v-list-item-icon v-bind="attrs" v-on="on">
                                <v-icon>mdi-cog</v-icon>
                            </v-list-item-icon>

                            <v-list-item-content v-bind="attrs" v-on="on">
                                <v-list-item-title>Video: {{videoResolution}}</v-list-item-title>
                            </v-list-item-content>
                        </template>
                        <v-list>
                            <!-- https://github.com/pion/ion-sdk-js/blob/master/src/stream.ts#L10 -->
                            <v-list-item
                                v-for="(item, i) in ['qvga', 'vga', 'shd', 'hd', 'fhd', 'qhd']"
                                :disabled="item == videoResolution"
                                :key="i"
                                @click="saveVideoResolution(item)"
                            >
                                <v-list-item-title>{{ item }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </v-list-item>
                <v-list-item
                        v-for="item in getAppBarItems()"
                        :key="item.title"
                        @click="item.clickFunction"
                >
                    <v-list-item-icon>
                        <v-icon :color="item.color">{{ item.icon }}</v-icon>
                    </v-list-item-icon>

                    <v-list-item-content>
                        <v-list-item-title>{{ item.title }}</v-list-item-title>
                    </v-list-item-content>
                </v-list-item>
            </v-list>
        </v-navigation-drawer>


        <v-app-bar
                :color="wsConnected ? 'indigo' : 'error'"
                dark
                app
                id="myAppBar"
                :clipped-left="true"
        >
            <v-app-bar-nav-icon @click="toggleLeftNavigation"></v-app-bar-nav-icon>
            <v-btn v-if="showHangButton && !shareScreen && $vuetify.breakpoint.smAndUp" icon @click="shareScreenStart()"><v-icon>mdi-monitor-screenshot</v-icon></v-btn>
            <v-btn v-if="showHangButton && shareScreen" icon @click="shareScreenStop()"><v-icon>mdi-stop</v-icon></v-btn>
            <v-badge
                v-if="showCallButton || showHangButton"
                :content="videoChatUsersCount"
                :value="videoChatUsersCount"
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
            <v-spacer></v-spacer>
            <v-tooltip bottom v-if="!wsConnected">
                <template v-slot:activator="{ on, attrs }">
                    <v-icon color="white" class="mr-2" v-bind="attrs" v-on="on">mdi-lan-disconnect</v-icon>
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
                <v-snackbar v-model="showWebsocketRestored" color="black" timeout="-1" :multi-line="true">
                    Websocket connection has been restored, press to update
                    <template v-slot:action="{ attrs }">
                        <v-btn
                            text
                            v-bind="attrs"
                            @click="onPressWebsocketRestored()"
                        >
                            Update
                        </v-btn>
                        <v-btn text v-bind="attrs" @click="showWebsocketRestored = false">Close</v-btn>

                    </template>
                </v-snackbar>
                <v-snackbar v-model="invitedVideoChatAlert" class="call-blink" color="success" timeout="-1" :multi-line="true" :key="callReblinkCounter" top>
                    You are called into chat #{{invitedVideoChatId}}, press to join
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
        GET_MUTE_AUDIO,
        GET_MUTE_VIDEO, GET_SEARCH_STRING,
        GET_SHARE_SCREEN,
        GET_SHOW_CALL_BUTTON,
        GET_SHOW_CHAT_EDIT_BUTTON,
        GET_SHOW_HANG_BUTTON,
        GET_SHOW_SEARCH, GET_TITLE,
        GET_USER,
        GET_VIDEO_CHAT_USERS_COUNT, SET_SEARCH_STRING,
        UNSET_USER
    } from "./store";
    import bus, {
        AUDIO_START_MUTING,
        VIDEO_START_MUTING,
        CHANGE_WEBSOCKET_STATUS,
        LOGGED_OUT,
        OPEN_CHAT_EDIT,
        OPEN_INFO_DIALOG,
        OPEN_PERMISSIONS_WARNING_MODAL,
        SHARE_SCREEN_START, SHARE_SCREEN_STOP,
        VIDEO_CALL_INVITED, REFRESH_ON_WEBSOCKET_RESTORED, REQUEST_CHANGE_VIDEO_RESOLUTION, VIDEO_RESOLUTION_CHANGED,
    } from "./bus";
    import ChatEdit from "./ChatEdit";
    import debounce from "lodash/debounce";
    import {chat_name, profile_self_name, chat_list_name, videochat_name} from "./routes";
    import SimpleModal from "./SimpleModal";
    import ChooseAvatar from "./ChooseAvatar";
    import {getCorrectUserAvatar} from "./utils";
    import ChatParticipants from "./ChatParticipants";
    import PermissionsWarning from "./PermissionsWarning";

    const audio = new Audio("/call.mp3");

    export default {
        data () {
            return {
                appBarItems: [
                    { title: 'Unmute audio', icon: 'mdi-microphone-off', color: 'error', clickFunction: this.toggleMuteAudio, requireAuthenticated: true, displayCondition: this.shouldDisplayAudioUnmute},
                    { title: 'Mute audio', icon: 'mdi-microphone', color: 'primary', clickFunction: this.toggleMuteAudio, requireAuthenticated: true, displayCondition: this.shouldDisplayAudioMute},
                    { title: 'Unmute video', icon: 'mdi-video-off', color: 'error', clickFunction: this.toggleMuteVideo, requireAuthenticated: true, displayCondition: this.shouldDisplayVideoUnmute},
                    { title: 'Mute video', icon: 'mdi-video', color: 'primary', clickFunction: this.toggleMuteVideo, requireAuthenticated: true, displayCondition: this.shouldDisplayVideoMute},

                    { title: 'Chats', icon: 'mdi-home-city', clickFunction: this.goHome, requireAuthenticated: false },
                    { title: 'New chat', icon: 'mdi-plus-circle-outline', clickFunction: this.createChat, requireAuthenticated: true},
                    { title: 'Edit chat', icon: 'mdi-lead-pencil', clickFunction: this.editChat, requireAuthenticated: true, displayCondition: this.shouldDisplayEditChat},
                    { title: 'My Account', icon: 'mdi-account', clickFunction: this.goProfile, requireAuthenticated: true },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout, requireAuthenticated: true },
                ],
                drawer: this.$vuetify.breakpoint.lgAndUp,
                lastError: "",
                showAlert: false,
                wsConnected: false,
                invitedVideoChatId: 0,
                invitedVideoChatAlert: false,
                callReblinkCounter: 0,
                showWebsocketRestored: false,
                videoResolution: null,
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
            doSearch(searchString) {
                this.$store.commit(SET_SEARCH_STRING, searchString);
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
            onChangeWsStatus({connected, wasInitialized}) {
                console.log("onChangeWsStatus: connected", connected, "wasInitialized", wasInitialized)
                this.wsConnected = connected;
                if (connected && wasInitialized) {
                    this.showWebsocketRestored = true;
                }
            },
            onInfoClicked() {
                bus.$emit(OPEN_INFO_DIALOG, this.chatId);
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
            isVideoRoute() {
                return this.$route.name == videochat_name
            },
            toggleMuteAudio() {
                bus.$emit(AUDIO_START_MUTING, !this.audioMuted)
            },
            toggleMuteVideo() {
                bus.$emit(VIDEO_START_MUTING, !this.videoMuted)
            },
            shouldDisplayAudioUnmute() {
                return this.isVideoRoute() && this.audioMuted;
            },
            shouldDisplayAudioMute() {
                return this.isVideoRoute() && !this.audioMuted;
            },
            shouldDisplayVideoUnmute() {
                return !this.shareScreen && this.isVideoRoute() && this.videoMuted;
            },
            shouldDisplayVideoMute() {
                return !this.shareScreen && this.isVideoRoute() && !this.videoMuted;
            },
            shouldDisplayEditChat() {
                return this.showChatEditButton;
            },
            onPressWebsocketRestored() {
                this.showWebsocketRestored = false;
                bus.$emit(REFRESH_ON_WEBSOCKET_RESTORED);
            },
            saveVideoResolution(newResolution) {
                console.log("Saving new video resolution", newResolution);
                bus.$emit(REQUEST_CHANGE_VIDEO_RESOLUTION, newResolution);
            },
            onVideoResolutionChanged(res) {
                this.videoResolution = res;
            },
        },
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
                videoMuted: GET_MUTE_VIDEO,
                audioMuted: GET_MUTE_AUDIO,
                showCallButton: GET_SHOW_CALL_BUTTON,
                showHangButton: GET_SHOW_HANG_BUTTON,
                shareScreen: GET_SHARE_SCREEN,
                videoChatUsersCount: GET_VIDEO_CHAT_USERS_COUNT,
                showChatEditButton: GET_SHOW_CHAT_EDIT_BUTTON,
                showSearch: GET_SHOW_SEARCH,
                chatId: GET_CHAT_ID,
                title: GET_TITLE,
                chatUsersCount: GET_CHAT_USERS_COUNT,
            }), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                return getCorrectUserAvatar(this.currentUser.avatar);
            },
            searchChatString: {
                get() {
                    return this.$store.getters[GET_SEARCH_STRING];
                },
                set(newValue) {
                    this.doSearch(newValue);
                }
            }
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(CHANGE_WEBSOCKET_STATUS, this.onChangeWsStatus);
            bus.$on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
            bus.$on(VIDEO_RESOLUTION_CHANGED, this.onVideoResolutionChanged)
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