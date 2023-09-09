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
      <v-list-item @click.prevent="goHome()" :href="getRouteRoot()" prepend-icon="mdi-home" :title="$vuetify.locale.t('$vuetify.start')"></v-list-item>
      <v-list-item @click.prevent="goChats()" :href="getRouteChats()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
      <v-list-item @click.prevent="openFiles()" prepend-icon="mdi-file-download" :title="$vuetify.locale.t('$vuetify.files')"></v-list-item>
      <v-list-item @click.prevent="onNotificationsClicked()">
        <template v-slot:prepend>
            <v-badge
                :content="notificationsCount"
                :model-value="showNotificationBadge"
                color="red"
                offset-y="-2"
                offset-x="-6"
                class="notifications-badge"
            >
                <v-icon class="notification-icon">mdi-bell</v-icon>
            </v-badge>
        </template>
        <template v-slot:title>
            {{ $vuetify.locale.t('$vuetify.notifications') }}
        </template>
      </v-list-item>
      <v-list-item @click.prevent="openSettings()" prepend-icon="mdi-cog" :title="$vuetify.locale.t('$vuetify.settings')"></v-list-item>
      <v-list-item :disabled="loading" @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {chat_list_name, chat_name, chats, profile, profile_self_name, root, root_name} from "@/router/routes";
import axios from "axios";
import bus, {LOGGED_OUT, OPEN_NOTIFICATIONS_DIALOG, OPEN_SETTINGS, OPEN_VIEW_FILES_DIALOG} from "@/bus/bus";
import {goToPreserving} from "@/mixins/searchString";

export default {
  data() {
      return {
          loading: false
      }
  },
  computed: {
    ...mapStores(useChatStore),
      notificationsCount() {
          return this.chatStore.notifications.length
      },
      showNotificationBadge() {
          return this.notificationsCount != 0
      },
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
    goHome() {
      this.$router.push({name: root_name} )
    },
    getRouteChats() {
      return chats
    },
    goChats() {
      goToPreserving(this.$route, this.$router, { name: chat_list_name});
    },
    getRouteProfile() {
      return profile
    },
    openSettings() {
      bus.emit(OPEN_SETTINGS)
    },
    logout(){
      this.loading = true;
      axios.post(`/api/logout`).then(() => {
        this.chatStore.unsetUser();
        bus.emit(LOGGED_OUT, null);
      }).finally(()=>{
          this.loading = false;
      });
    },
    shouldDisplayLogout() {
      return this.chatStore.currentUser != null;
    },
    onNotificationsClicked() {
      bus.emit(OPEN_NOTIFICATIONS_DIALOG);
    },
    openFiles() {
      bus.emit(OPEN_VIEW_FILES_DIALOG, { });
    },
  }
}
</script>

<style lang="scss">
@use './styles/settings';

.notifications-badge {

    .notification-icon {
        opacity: settings.$list-item-icon-opacity;
    }
}
</style>
