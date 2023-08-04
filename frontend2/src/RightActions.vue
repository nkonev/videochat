<template>
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
      <v-list-item @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {chat_list_name, profile, profile_self_name, root} from "@/router/routes";
import axios from "axios";
import bus, {LOGGED_OUT} from "@/bus/bus";

export default {
  computed: {
    ...mapStores(useChatStore),
  },
  methods: {
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
    },
    goHome() {
      this.$router.push(({ name: chat_list_name}))
    },
    logout(){
      console.log("Logout");
      axios.post(`/api/logout`).then(() => {
        this.chatStore.unsetUser();
        bus.emit(LOGGED_OUT, null);
      });
    },
    shouldDisplayLogout() {
      return this.chatStore.currentUser != null;
    },
  }
}
</script>
