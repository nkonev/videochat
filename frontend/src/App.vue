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
                        <img :src="currentUser.avatar">
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
                        v-for="item in appBarItems"
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

            <v-spacer></v-spacer>
            <v-toolbar-title>Chats</v-toolbar-title>
            <v-spacer></v-spacer>

            <v-btn icon>
                <v-icon>mdi-magnify</v-icon>
            </v-btn>
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

                <v-card
                        max-width="1000"
                        class="mx-auto"
                >
                    <EditChat v-model="openEditModal" :editChatId="editChatId"/>
                    <LoginModal/>

                    <v-list>
                            <v-list-item
                                    v-for="(item, index) in chats"
                                    :key="item.id"
                                    @click=""
                            >
                                <v-list-item-content>
                                    <v-list-item-title v-html="item.name"></v-list-item-title>
                                    <v-list-item-subtitle v-html="item.participantIds"></v-list-item-subtitle>
                                </v-list-item-content>
                                <v-list-item-action>
                                    <v-btn color="primary" fab dark small @click="editChat(item)"><v-icon dark>mdi-plus</v-icon></v-btn>
                                </v-list-item-action>
                            </v-list-item>
                    </v-list>
                    <infinite-loading @infinite="infiniteHandler" :identifier="infiniteId"></infinite-loading>

                </v-card>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import axios from 'axios';
    import EditChat from "./EditChat";
    import InfiniteLoading from 'vue-infinite-loading';
    import LoginModal from "./LoginModal";
    import {mapGetters} from 'vuex'
    import {FETCH_USER_PROFILE, GET_USER, UNSET_USER} from "./store";
    import bus, {CHAT_SAVED, LOGGED_IN, LOGGED_OUT} from "./bus";

    const replaceInArray = (array, element) => {
        const foundIndex = array.findIndex(value => value.id === element.id);
        if (foundIndex === -1) {
            return false;
        } else {
            array[foundIndex] = element;
            return true;
        }
    };

    export default {
        data () {
            return {
                page: 0,
                chats: [],
                openEditModal: false,
                editChatId: null,
                appBarItems: [
                    { title: 'Home', icon: 'mdi-home-city', clickFunction: ()=>{} },
                    { title: 'My Account', icon: 'mdi-account', clickFunction: ()=>{} },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout },
                ],
                drawer: true,
                infiniteId: new Date(),
                lastError: "",
                showAlert: false,
            }
        },
        components:{
            EditChat,
            InfiniteLoading,
            LoginModal
        },
        methods:{
            createChat() {
                this.$data.editChatId = null;
                this.$data.openEditModal = true;
            },
            editChat(chat) {
                const chatId = chat.id;
                console.log("Will add participants to chat", chatId);
                this.$data.editChatId = chatId;
                this.$data.openEditModal = true;
            },
            infiniteHandler($state) {
                axios.get(`/api/chat`, {
                    params: {
                        page: this.page,
                    },
                }).then(({ data }) => {
                    if (data.length) {
                        this.page += 1;
                        this.chats.push(...data);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
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
            reloadChats() {
                this.infiniteId += 1;
                console.log("Resetting infinite loader", this.infiniteId);
            },
            onError(errText){
                this.showAlert = true;
                this.lastError = errText;
            },
            rerenderChat(dto) {
                console.log("Rerendering chat", dto);
                //const chatIndex = this.chats.findIndex(value => value.id === dto.id);
                //console.log("Found chat", chatIndex);
                const replaced = replaceInArray(this.chats, dto);
                if (!replaced) {
                    // reload last page
                    axios.get(`/api/chat`, {
                        params: {
                            page: this.page,
                        },
                    }).then(({ data }) => {
                        if (data.length) {
                            // TODO Array.prototype.splice() and lastPageActualSize
                            data.forEach((element, index) => {
                                replaceInArray(this.chats, element)
                            });
                        } else {
                            // if no data on current page load previous
                            axios.get(`/api/chat`, {
                                params: {
                                    page: this.page - 1,
                                },
                            }).then(({ data }) => {
                                if (data.length) {
                                    // TODO Array.prototype.splice() and lastPageActualSize
                                    data.forEach((element, index) => {
                                        replaceInArray(this.chats, element)
                                    });
                                }
                            })
                        }
                    });
                    this.reloadChats();
                }
            },
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadChats);
            bus.$on(CHAT_SAVED, this.rerenderChat);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadChats);
            bus.$off(CHAT_SAVED, this.rerenderChat);
        },
    }
</script>

<style lang="stylus">
    @import '~typeface-roboto/index.css'
</style>