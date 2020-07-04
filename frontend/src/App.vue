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

            <v-text-field hide-details prepend-icon="mdi-magnify" single-line></v-text-field>
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
                <EditChat/>
                <router-view/>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
    import axios from 'axios';
    import LoginModal from "./LoginModal";
    import {mapGetters} from 'vuex'
    import {FETCH_USER_PROFILE, GET_USER, UNSET_USER} from "./store";
    import bus, {LOGGED_OUT, OPEN_CHAT_EDIT} from "./bus";
    import EditChat from "./EditChat";

    export default {
        data () {
            return {
                appBarItems: [
                    { title: 'Home', icon: 'mdi-home-city', clickFunction: ()=>{} },
                    { title: 'My Account', icon: 'mdi-account', clickFunction: ()=>{} },
                    { title: 'Logout', icon: 'mdi-logout', clickFunction: this.logout },
                ],
                drawer: true,
                lastError: "",
                showAlert: false,
            }
        },
        components:{
            LoginModal,
            EditChat
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
            onError(errText){
                this.showAlert = true;
                this.lastError = errText;
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
            }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
    }
</script>

<style lang="stylus">
    @import '~typeface-roboto/index.css'
</style>