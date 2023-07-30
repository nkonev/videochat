<template>
  <v-app>
      <v-navigation-drawer
          left
          app
          :clipped="true"
          v-model="drawer"
      >
          <v-list>
              <v-list-item v-if="chatStore.currentUser" @click.prevent="onProfileClicked()" link :href="require('./routes').profile"
                           :prepend-avatar="chatStore.currentUser.avatar"
                           :title="chatStore.currentUser.login"
                           :subtitle="chatStore.currentUser.shortInfo"
              ></v-list-item>
          </v-list>

          <v-divider></v-divider>

          <v-list density="compact" nav>
              <v-list-item @click.prevent="goHome()" :href="require('./routes').root" prepend-icon="mdi-forum" :title="$vuetify.lang.t('$vuetify.chats')"></v-list-item>
          </v-list>
      </v-navigation-drawer>


      <v-app-bar
          color='indigo'
          dark
          app
          id="myAppBar"
          :clipped-left="true"
          :dense="!isMobile()"
      >
          <v-badge
              :content="chatStore.notificationsCount"
              :value="chatStore.notificationsCount && !drawer"
              color="red"
              overlap
              offset-y="1.8em"
          >
              <v-app-bar-nav-icon @click="toggleLeftNavigation"></v-app-bar-nav-icon>
          </v-badge>

          <template v-if="showSearchButton || !isMobile()">
              <v-badge v-if="chatStore.showCallButton || chatStore.showHangButton"
                       style="padding-left: 10px"
                       :content="chatStore.videoChatUsersCount"
                       :value="chatStore.videoChatUsersCount"
                       color="green"
                       overlap
                       offset-y="1.8em"
              >
                  <v-btn v-if="chatStore.showCallButton" icon @click="createCall()" :title="chatStore.tetATet ? $vuetify.lang.t('$vuetify.call_up') : $vuetify.lang.t('$vuetify.enter_into_call')">
                      <v-icon :x-large="isMobile()" color="green">{{tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                  </v-btn>
                  <v-btn v-if="chatStore.showHangButton" icon @click="stopCall()" :title="$vuetify.lang.t('$vuetify.leave_call')">
                      <v-icon :x-large="isMobile()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'red--text'">mdi-phone</v-icon>
                  </v-btn>
              </v-badge>


          </template>
          <v-spacer></v-spacer>

      </v-app-bar>

    <v-main>
      <v-container fluid class="ma-0 pa-0" style="height: 100%">

          <LoginModal/>

          <router-view />
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import '@fontsource/roboto';
import { hasLength } from "@/utils";
import {chat_list_name, chat_name, profile_self_name, videochat_name} from "@/routes";
import axios from "axios";
import bus, {LOGGED_OUT} from "@/bus";
import LoginModal from "@/LoginModal.vue";
import vuetify from "@/plugins/vuetify";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'

export default {
    data() {
        return {
            drawer: !vuetify.display.mobile,
            lastAnswered: 0,
            showSearchButton: true,
        }
    },
    computed: {
        // https://pinia.vuejs.org/cookbook/options-api.html#usage-without-setup
        ...mapStores(useChatStore),
        currentUserAvatar() {
            return this.chatStore.currentUser?.avatar;
        },
        // it differs from original
        chatId() {
            return this.$route.params.id
        },
    },
    methods: {
        showCurrentUserSubtitle(){
            return hasLength(this.chatStore.currentUser?.shortInfo)
        },
        goHome() {
            this.$router.push(({ name: chat_list_name}))
        },
        toggleLeftNavigation() {
            this.$data.drawer = !this.$data.drawer;
        },
        logout(){
            console.log("Logout");
            axios.post(`/api/logout`).then(({ data }) => {
                this.chatStore.unsetUser();
                bus.emit(LOGGED_OUT, null);
            });
        },
        createCall() {
            console.debug("createCall");
            axios.put(`/api/video/${this.chatId}/dial/start`).then(()=>{
                const routerNewState = { name: videochat_name};
                // this.navigateToWithPreservingSearchStringInQuery(routerNewState); // TODO
                this.updateLastAnsweredTimestamp();
            })
        },
        stopCall() {
            console.debug("stopping Call");
            const routerNewState = { name: chat_name, params: { leavingVideoAcceptableParam: true } };
            // this.navigateToWithPreservingSearchStringInQuery(routerNewState); // TODO
            this.updateLastAnsweredTimestamp();
        },
        updateLastAnsweredTimestamp() {
            this.lastAnswered = +new Date();
        },
        goProfile() {
            this.$router.push(({ name: profile_self_name}))
        },
        onProfileClicked() {
            if (!this.isMobile()) {
                this.goProfile();
            }
        },
    },
    components: {
        LoginModal
    },
}
</script>
