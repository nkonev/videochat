<template>
    <v-list>
      <v-list-item id="test-user-login" v-if="chatStore.currentUser"
                   :prepend-avatar="chatStore.currentUser.avatar"
                   :title="chatStore.currentUser.login"
                   :subtitle="chatStore.currentUser.shortInfo"
      ></v-list-item>
    </v-list>

    <v-divider></v-divider>

    <v-list density="compact" nav>
      <v-list-item v-if="shouldShowUpperChats()" @click.prevent="goChats()" :href="getRouteChats()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
      <v-list-item v-if="shouldDisplayCopyCallLink()" @click.prevent="copyCallLink()" prepend-icon="mdi-content-copy" :title="$vuetify.locale.t('$vuetify.copy_video_call_link')"></v-list-item>
      <v-list-item v-if="shouldDisplayAddVideoSource()" @click.prevent="addVideoSource()" prepend-icon="mdi-video-plus" :title="$vuetify.locale.t('$vuetify.source_add')"></v-list-item>
      <v-list-item v-if="shouldShowHome()" @click.prevent="goHome()" :href="getRouteRoot()" prepend-icon="mdi-home" :title="$vuetify.locale.t('$vuetify.start')"></v-list-item>
      <v-list-item v-if="shouldShowLowerChats()" @click.prevent="goChats()" :href="getRouteChats()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
      <v-list-item @click.prevent="goBlogs()" :href="getRouteBlogs()" prepend-icon="mdi-postage-stamp" :title="$vuetify.locale.t('$vuetify.blogs')"></v-list-item>
      <v-list-item v-if="shouldDisplayCreateChat()" @click="createChat()" prepend-icon="mdi-plus" id="test-new-chat-dialog-button" :title="$vuetify.locale.t('$vuetify.new_chat')"></v-list-item>
      <v-list-item @click="editChat()" v-if="shouldDisplayEditChat()" prepend-icon="mdi-lead-pencil" :title="$vuetify.locale.t('$vuetify.edit_chat')"></v-list-item>
      <v-list-item v-if="shouldDisplayCopyCallLinkDesktop()" @click.prevent="copyCallLink()" prepend-icon="mdi-content-copy" :title="$vuetify.locale.t('$vuetify.copy_video_call_link')"></v-list-item>
      <v-list-item v-if="canShowFiles()" @click.prevent="openFiles()" prepend-icon="mdi-file-download" :title="$vuetify.locale.t('$vuetify.files')"></v-list-item>
      <v-list-item @click="openPinnedMessages()" v-if="shouldPinnedMessages()" prepend-icon="mdi-pin" :title="$vuetify.locale.t('$vuetify.pinned_messages')"></v-list-item>
      <v-list-item @click.prevent="onNotificationsClicked()" v-if="shouldDisplayNotifications()">
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
      <v-list-item @click.prevent="openUsers()" :href="getRouteUsers()" prepend-icon="mdi-account-group" :title="$vuetify.locale.t('$vuetify.users')"></v-list-item>
      <v-list-item @click.prevent="openSettings()" prepend-icon="mdi-cog" :title="$vuetify.locale.t('$vuetify.settings')"></v-list-item>
      <v-list-item :disabled="loading" @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {
    blog,
    chat_list_name,
    chats,
    profile_list_name,
    profiles,
    root,
    root_name, videochat_name
} from "@/router/routes";
import axios from "axios";
import bus, {
    ADD_VIDEO_SOURCE_DIALOG,
    LOGGED_OUT, OPEN_CHAT_EDIT,
    OPEN_NOTIFICATIONS_DIALOG,
    OPEN_PINNED_MESSAGES_MODAL,
    OPEN_SETTINGS,
    OPEN_VIEW_FILES_DIALOG
} from "@/bus/bus";
import {goToPreserving} from "@/mixins/searchString";
import {copyCallLink, hasLength, isChatRoute} from "@/utils";

export default {
  data() {
      return {
          loading: false
      }
  },
  computed: {
    ...mapStores(useChatStore),
      notificationsCount() {
          return this.chatStore.notificationsCount
      },
      showNotificationBadge() {
          return this.notificationsCount != 0
      },
      chatId() {
        return this.$route.params.id
      },
  },
  methods: {
    createChat() {
      bus.emit(OPEN_CHAT_EDIT, null);
    },
    editChat() {
      bus.emit(OPEN_CHAT_EDIT, this.chatId);
    },
    shouldDisplayCopyCallLink() {
      return (this.chatStore.showCallButton || this.chatStore.showHangButton) && this.isMobile()
    },
    shouldDisplayCopyCallLinkDesktop() {
      return (this.chatStore.showCallButton || this.chatStore.showHangButton) && !this.isMobile()
    },
    shouldDisplayAddVideoSource() {
      return this.$route.name == videochat_name && this.isMobile()
    },
    shouldDisplayCreateChat() {
      return this.chatStore.currentUser
    },
    shouldDisplayEditChat() {
      return this.chatStore.currentUser && this.chatStore.showChatEditButton;
    },
    getRouteRoot() {
      return root
    },
    goHome() {
      this.$router.push({name: root_name} )
    },
    copyCallLink() {
      copyCallLink(this.chatId)
    },
    addVideoSource() {
          bus.emit(ADD_VIDEO_SOURCE_DIALOG);
    },
    getRouteChats() {
      return chats
    },
    getRouteBlogs() {
      return blog
    },
    getRouteUsers() {
      return profiles;
    },
    goChats() {
      goToPreserving(this.$route, this.$router, { name: chat_list_name});
    },
    goBlogs() {
      window.location.href = blog
    },
    openUsers() {
      goToPreserving(this.$route, this.$router, { name: profile_list_name});
    },
    canShowFiles() {
      return this.chatStore.currentUser && hasLength(this.chatId);
    },
    openSettings() {
      bus.emit(OPEN_SETTINGS)
    },
    logout(){
      this.loading = true;
      axios.post(`/api/aaa/logout`).then(() => {
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
    openPinnedMessages() {
      bus.emit(OPEN_PINNED_MESSAGES_MODAL);
    },
    shouldPinnedMessages() {
      return this.chatStore.currentUser && hasLength(this.chatId);
    },
    shouldDisplayNotifications() {
      return this.chatStore.currentUser
    },
    shouldShowHome() {
      if (!this.isMobile()) {
        return true
      } else {
        return !isChatRoute(this.$route)
      }
    },
    shouldShowUpperChats() {
        if (this.isMobile()) {
            return isChatRoute(this.$route)
        } else {
            return false
        }
    },
    shouldShowLowerChats() {
        if (!this.isMobile()) {
            return true
        } else {
            return !isChatRoute(this.$route)
        }
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
