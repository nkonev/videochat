<template>
  <v-app>

      <v-app-bar
          color='indigo'
          id="myAppBar"
          :density="getDensity()"
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
                  <v-btn v-if="chatStore.showCallButton" icon @click="createCall()" :title="chatStore.tetATet ? $vuetify.locale.t('$vuetify.call_up') : $vuetify.locale.t('$vuetify.enter_into_call')">
                      <v-icon :x-large="isMobile()" color="green">{{tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                  </v-btn>
                  <v-btn v-if="chatStore.showHangButton" icon @click="stopCall()" :title="$vuetify.locale.t('$vuetify.leave_call')">
                      <v-icon :x-large="isMobile()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'red--text'">mdi-phone</v-icon>
                  </v-btn>
              </v-badge>


          </template>
          <v-spacer></v-spacer>

          <v-card variant="plain" min-width="400px" v-if="chatStore.isShowSearch">
              <v-text-field density="compact" variant="solo" :autofocus="isMobile()" prepend-inner-icon="mdi-magnify" hide-details single-line @input="clearRouteHash()" v-model="searchString" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
          </v-card>

      </v-app-bar>

      <v-navigation-drawer
          v-model="drawer"
          width="400"
      >
          <v-list>
              <v-list-item v-if="chatStore.currentUser" @click.prevent="onProfileClicked()" link :href="getRouteProfile()"
                           :prepend-avatar="chatStore.currentUser.avatar"
                           :title="chatStore.currentUser.login"
                           :subtitle="chatStore.currentUser.shortInfo"
              ></v-list-item>
          </v-list>

          <v-divider></v-divider>

          <v-list density="compact" nav>
              <v-list-item @click.prevent="goHome()" :href="getRouteRoot()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
          </v-list>
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

    <v-navigation-drawer location="right" v-model="drawer">
      <v-list>
        <v-list-item
          v-for="n in 5"
          :key="n"
          :title="`Item ${ n }`"
          prepend-icon="mdi-forum"
          link
        >
        </v-list-item>
      </v-list>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import '@fontsource/roboto';
import { hasLength } from "@/utils";
import {chat_list_name, chat_name, profile, profile_self_name, root, videochat_name} from "@/router/routes";
import axios from "axios";
import bus, {LOGGED_OUT, SEARCH_STRING_CHANGED} from "@/bus/bus";
import LoginModal from "@/LoginModal.vue";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'
import searchString from "@/mixins/searchString";

export default {
    mixins: [
      searchString()
    ],
    data() {
        return {
            drawer: !this.isMobile(),
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
        getRouteRoot() {
            return root
        },
        getRouteProfile() {
            return profile
        }
    },
    components: {
        LoginModal
    },
    created() {
        this.chatStore.fetchAvailableOauth2Providers().then(() => {
            this.chatStore.fetchUserProfile();
        })
    },
    watch: {
      '$route.query.q': {
        handler: function (newValue, oldValue) {
          console.debug("Route q", oldValue, "->", newValue);
          bus.emit(SEARCH_STRING_CHANGED, {oldValue: oldValue, newValue: newValue});
        },
      },
    }
}
</script>
