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
                <v-list-item two-line v-if="currentUser" @click="openAvatarDialog()">
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

            <v-btn icon @click="createChat">
                <v-icon>mdi-plus-circle-outline</v-icon>
            </v-btn>
            <v-btn v-if="showChatEditButton" icon @click="editChat">
                <v-icon>mdi-lead-pencil</v-icon>
            </v-btn>
            <v-btn v-if="showCallButton" icon @click="createCall">
                <v-icon color="green">mdi-phone</v-icon>
            </v-btn>
            <v-btn v-if="showHangButton" icon @click="stopCall">
                <v-icon color="red">mdi-phone</v-icon>
            </v-btn>

            <v-spacer></v-spacer>
            <v-toolbar-title>{{title}}</v-toolbar-title>
            <v-spacer></v-spacer>
            <v-tooltip bottom v-if="!wsConnected">
                <template v-slot:activator="{ on, attrs }">
                    <v-icon color="red" v-bind="attrs" v-on="on">mdi-lan-disconnect</v-icon>
                </template>
                <span>Websocket not connected</span>
            </v-tooltip>
            <v-card light v-if="showSearch">
                <v-text-field prepend-icon="mdi-magnify" hide-details single-line v-model="searchChatString" clearable clear-icon="mdi-close-circle"></v-text-field>
            </v-card>
        </v-app-bar>


        <v-main>
            <v-container>
                <v-alert
                        dismissible
                        v-model="showAlert"
                        prominent
                        type="error"
                >
                    <v-row align="center">
                        <v-col class="grow">{{lastError}}</v-col>
                    </v-row>
                </v-alert>
                <LoginModal/>
                <ChatEdit/>
                <ChatDelete/>
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
        CHANGE_PHONE_BUTTON,
        CHANGE_TITLE, CHANGE_WEBSOCKET_STATUS,
        LOGGED_OUT,
        OPEN_CHAT_EDIT,
        OPEN_CHOOSE_AVATAR,
    } from "./bus";
    import ChatEdit from "./ChatEdit";
    import debounce from "lodash/debounce";
    import {chat_name, profile_name, root_name, videochat_name} from "./routes";
    import ChatDelete from "./ChatDelete";
    import ChooseAvatar from "./ChooseAvatar";
    import {getCorrectUserAvatar} from "./utils";

    export default {
        data () {
            return {
                title: "",
                appBarItems: [
                    { title: 'Chats', icon: 'mdi-home-city', clickFunction: this.goHome, requireAuthenticated: false },
                    { title: 'My Account', icon: 'mdi-account', clickFunction: this.goProfile, requireAuthenticated: true },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout, requireAuthenticated: true },
                ],
                drawer: true,
                lastError: "",
                showAlert: false,
                searchChatString: "",
                showSearch: false,
                showChatEditButton: false,
                showCallButton: false,
                showHangButton: false,
                chatEditId: null,
                wsConnected: false
            }
        },
        components:{
            LoginModal,
            ChatEdit,
            ChatDelete,
            ChooseAvatar
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
                this.$router.push(({ name: root_name}))
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
                })
            },
            changeTitle({title, isShowSearch, isShowChatEditButton, chatEditId}) {
                this.title = title;
                this.showSearch = isShowSearch;
                this.showChatEditButton = isShowChatEditButton;
                this.chatEditId = chatEditId;
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
            createCall() {
                console.log("createCall");
                this.$router.push({ name: videochat_name});
            },
            stopCall() {
                console.log("stopCall");
                this.$router.push({ name: chat_name});
            },
            openAvatarDialog() {
                bus.$emit(OPEN_CHOOSE_AVATAR);
            },
            onChangeWsStatus(value) {
                this.wsConnected = value;
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
            currentUserAvatar() {
                console.log("Invoke avatar getter method");
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
            bus.$on(CHANGE_WEBSOCKET_STATUS, this.onChangeWsStatus)
        },
        watch: {
            searchChatString (searchString) {
                this.doSearch(searchString);
            },
        },

    }
</script>
