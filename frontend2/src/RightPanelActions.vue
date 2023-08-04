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
      <v-list-item @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {chat_list_name, chat_name, profile, profile_self_name, root} from "@/router/routes";
import axios from "axios";
import bus, {LOGGED_OUT, OPEN_NOTIFICATIONS_DIALOG} from "@/bus/bus";
import {goToPreserving} from "@/mixins/searchString";

export default {
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
    getRouteProfile() {
      return profile
    },
    goHome() {
      goToPreserving(this.$route, this.$router, { name: chat_list_name});
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
    onNotificationsClicked() {
      bus.emit(OPEN_NOTIFICATIONS_DIALOG);
    },
  }
}
</script>

<style lang="scss">
@use './styles/settings';

.notifications-badge {
    margin-inline-end: settings.$list-item-icon-margin-end;

    .notification-icon {
        opacity: settings.$list-item-icon-opacity;
    }
}
</style>
