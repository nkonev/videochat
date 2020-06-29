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

            <v-btn icon @click="openModal">
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
                <v-card
                        max-width="1000"
                        class="mx-auto"
                >
                    <EditChat v-model="openEditModal"/>
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
    import bus, {LOGGED_IN, LOGGED_OUT, UNAUTHORIZED} from "./bus";

    export default {
        data () {
            return {
                page: 0,
                chats: [],
                openEditModal: false,
                appBarItems: [
                    { title: 'Home', icon: 'mdi-home-city', clickFunction: ()=>{} },
                    { title: 'My Account', icon: 'mdi-account', clickFunction: ()=>{} },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout },
                ],
                drawer: true,
                infiniteId: new Date(),
            }
        },
        components:{
            EditChat,
            InfiniteLoading,
            LoginModal
        },
        methods:{
            openModal() {
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
                // this.$data.page = 0;
                // this.$data.chats = [];
                this.$nextTick(() => {
                    this.infiniteId += 1;
                    console.log("Resetting infinite loader");
                })
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            bus.$on(LOGGED_IN, this.reloadChats);
            bus.$on(LOGGED_OUT, this.reloadChats);
        },
        destroyed() {
            bus.$off(LOGGED_IN, this.reloadChats);
            bus.$off(LOGGED_OUT, this.reloadChats);
        },
    }
</script>

<style lang="stylus" scoped>
    .application {
        font-family: Arial, sans-serif;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;


        #input-usage .v-input__prepend-outer,
        #input-usage .v-input__append-outer,
        #input-usage .v-input__slot,
        #input-usage .v-messages {
            border: 1px dashed rgba(0,0,0, .4);
        }
    }
</style>
