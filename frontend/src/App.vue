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
                <v-icon>{{mdiPlusCircleOutline}}</v-icon>
            </v-btn>

            <v-spacer></v-spacer>
            <v-toolbar-title>{{title}}</v-toolbar-title>
            <v-spacer></v-spacer>

            <v-card light v-if="showSearch">
                <v-text-field :prepend-icon="mdiMagnify" hide-details single-line v-model="searchChatString"></v-text-field>
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
    import bus, {CHANGE_TITLE, LOGGED_OUT, OPEN_CHAT_EDIT} from "./bus";
    import ChatEdit from "./ChatEdit";
    import debounce from "lodash/debounce";
    import {root_name} from "./routes";
    import ChatDelete from "./ChatDelete";

    import Vue from 'vue'
    import { library } from '@fortawesome/fontawesome-svg-core'
    import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
    import { faFacebook, faVk } from '@fortawesome/free-brands-svg-icons'
    library.add(faFacebook, faVk);
    Vue.component('font-awesome-icon', FontAwesomeIcon) // Register component globally

    import { mdiPlusCircleOutline, mdiHomeCity, mdiAccount, mdiLogout, mdiMagnify } from '@mdi/js'

    export default {
        data () {
            return {
                title: "",
                appBarItems: [
                    { title: 'Chats', icon: mdiHomeCity, clickFunction: this.goHome, requireAuthenticated: false },
                    { title: 'My Account', icon: mdiAccount, clickFunction: ()=>{}, requireAuthenticated: true },
                    { title: 'Logout', icon: mdiLogout, clickFunction: this.logout, requireAuthenticated: true },
                ],
                drawer: true,
                lastError: "",
                showAlert: false,
                searchChatString: "",
                showSearch: false,

                mdiPlusCircleOutline,
                mdiMagnify,
            }
        },
        components:{
            LoginModal,
            ChatEdit,
            ChatDelete
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
            onError(errText){
                this.showAlert = true;
                this.lastError = errText;
            },
            createChat() {
                bus.$emit(OPEN_CHAT_EDIT, null);
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
            changeTitle(newtitle, isShowSearch) {
                this.title = newtitle;
                this.showSearch = isShowSearch;
            },
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
        mounted() {
            this.$store.dispatch(FETCH_USER_PROFILE);
        },
        created() {
            this.doSearch = debounce(this.doSearch, 700);
            bus.$on(CHANGE_TITLE, this.changeTitle);
        },
        watch: {
            searchChatString (searchString) {
                this.doSearch(searchString);
            },
        },

    }
</script>
