<template>
  <v-app>

      <v-app-bar
          color='indigo'
          id="myAppBar"
          :density="getDensity()"
      >
          <v-badge
              :content="notificationsCount"
              :model-value="showNotificationBadge"
              color="red"
              overlap
              offset-y="10"
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
                  <v-btn v-if="chatStore.showCallButton" icon @click="createCall()" :title="chatStore.tetATet ? $vuetify.locale.t('$vuetify.call_up') : $vuetify.locale.t('$vuetify.enter_into_call')">
                      <v-icon :x-large="isMobile()" color="green">{{tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                  </v-btn>
                  <v-btn v-if="chatStore.showHangButton" icon @click="stopCall()" :title="$vuetify.locale.t('$vuetify.leave_call')">
                      <v-icon :x-large="isMobile()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'red--text'">mdi-phone</v-icon>
                  </v-btn>
              </v-badge>


          </template>

          <v-card variant="plain" min-width="330" v-if="chatStore.isShowSearch" style="margin-left: 1.2em">
              <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line @input="clearRouteHash()" v-model="searchString" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput">
                  <template v-slot:append-inner>
                      <v-btn icon density="compact"><v-icon>mdi-magnify</v-icon></v-btn>
                  </template>
              </v-text-field>
          </v-card>

          <v-spacer></v-spacer>

      </v-app-bar>

      <v-navigation-drawer
          v-model="chatStore.drawer"
          width="400"
      >
          <LeftPanelChats/>
      </v-navigation-drawer>

      <v-main>
        <v-container fluid class="ma-0 pa-0" style="height: 100%">

          <v-snackbar v-model="chatStore.showAlert" :color="chatStore.errorColor" timeout="-1" :multi-line="true" :transition="false">
            {{ chatStore.lastError }}

            <template v-slot:actions>
              <v-btn
                text
                @click="refreshPage()"
              >
                Refresh
              </v-btn>

              <v-btn
                text
                @click="closeError()"
              >
                Close
              </v-btn>
            </template>
          </v-snackbar>



          <LoginModal/>

            <router-view />
        </v-container>
    </v-main>

    <v-navigation-drawer location="right" v-model="chatStore.drawer">
        <RightPanelActions/>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import '@fontsource/roboto';
import { hasLength } from "@/utils";
import { chat_name, videochat_name} from "@/router/routes";
import axios from "axios";
import bus, {LOGGED_OUT, PROFILE_SET, SEARCH_STRING_CHANGED} from "@/bus/bus";
import LoginModal from "@/LoginModal.vue";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'
import searchString from "@/mixins/searchString";
import RightPanelActions from "@/RightPanelActions.vue";
import LeftPanelChats from "@/LeftPanelChats.vue";

export default {
    mixins: [
      searchString()
    ],
    data() {
        return {
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
        notificationsCount() {
            return this.chatStore.notifications.length
        },
        showNotificationBadge() {
            return this.notificationsCount != 0 && !this.chatStore.drawer
        },
    },
    methods: {
        clearRouteHash() {

        },
        resetInput() {
          this.searchString = null
        },

        refreshPage() {
          location.reload();
        },

        getDensity() {
            return this.isMobile() ? "comfortable" : "compact";
        },
        showCurrentUserSubtitle(){
            return hasLength(this.chatStore.currentUser?.shortInfo)
        },
        toggleLeftNavigation() {
            this.chatStore.drawer = !this.chatStore.drawer;
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

        onProfileSet(){
            this.chatStore.fetchNotifications();
        },
        onLoggedOut() {
            this.resetVariables();
        },
        resetVariables() {
            this.chatStore.unsetNotifications();
        },
    },
    components: {
        LeftPanelChats,
        RightPanelActions,
        LoginModal,
    },
    created() {
        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);

        this.chatStore.fetchAvailableOauth2Providers().then(() => {
            this.chatStore.fetchUserProfile();
        })
    },

    beforeUnmount() {
        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
    },

    watch: {
      '$route.query.q': {
        handler: function (newValue, oldValue) {
          console.debug("Route q", oldValue, "->", newValue);
          bus.emit(SEARCH_STRING_CHANGED, {oldValue: oldValue, newValue: newValue});
        },
      },
      'chatStore.currentUser': function(newUserValue, oldUserValue) {
        console.debug("User new", newUserValue, "old" , oldUserValue);
        if (newUserValue && !oldUserValue) {
            bus.emit(PROFILE_SET);
        }
      }

    }
}
</script>
