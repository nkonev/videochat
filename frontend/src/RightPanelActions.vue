<template>
    <v-list>
      <v-list-item id="test-user-login" v-if="chatStore.currentUser"
      >
          <template v-slot:prepend>
              <v-avatar :image="chatStore.currentUser.avatar"></v-avatar>
          </template>
          <template v-slot:title>
              <span :style="getLoginColoredStyle(chatStore.currentUser)">{{ chatStore.currentUser.login }}</span>
          </template>
          <template v-slot:subtitle>
              <span>{{ chatStore.currentUser.shortInfo }}</span>
          </template>
      </v-list-item>
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
      <v-list-item v-if="canShowFiles()" @click.prevent="openFiles()" prepend-icon="mdi-file-download" :title="$vuetify.locale.t('$vuetify.files')"></v-list-item>
      <v-list-item @click="openPinnedMessages()" v-if="shouldPinnedMessages()" prepend-icon="mdi-pin" :title="$vuetify.locale.t('$vuetify.pinned_messages')"></v-list-item>
      <v-list-item @click="openPublishedMessages()" v-if="shouldPublishedMessages()" prepend-icon="mdi-export" :title="$vuetify.locale.t('$vuetify.published_messages')"></v-list-item>
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
      <v-list-item :disabled="isLoggingOut" @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {
    blog,
    chat_list_name, chat_name,
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
    OPEN_PINNED_MESSAGES_MODAL, OPEN_PUBLISHED_MESSAGES_MODAL,
    OPEN_SETTINGS,
    OPEN_VIEW_FILES_DIALOG
} from "@/bus/bus";
import {goToPreservingQuery} from "@/mixins/searchString";
import {copyCallLink, getLoginColoredStyle, hasLength, isChatRoute} from "@/utils";

export default {
  data() {
      return {
          isLoggingOut: false
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
    getLoginColoredStyle,
    createChat() {
      bus.emit(OPEN_CHAT_EDIT, null);
    },
    editChat() {
      bus.emit(OPEN_CHAT_EDIT, this.chatId);
    },
    shouldDisplayCopyCallLink() {
      return (this.chatStore.showCallManagement) && this.isMobile()
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
      goToPreservingQuery(this.$route, this.$router, { name: chat_list_name});
    },
    goBlogs() {
      window.location.href = blog
    },
    openUsers() {
      this.chatStore.incrementProgressCount();
      this.$router.push({name: profile_list_name} ).finally(()=>{
        this.chatStore.decrementProgressCount();
      })
    },
    isChatable() {
        return this.$route.name == chat_name || this.$route.name == videochat_name
    },
    canShowFiles() {
      return this.chatStore.currentUser && this.isChatable();
    },
    openSettings() {
      bus.emit(OPEN_SETTINGS)
    },
    logout(){
      this.isLoggingOut = true;
      this.chatStore.incrementProgressCount();
      axios.post(`/api/aaa/logout`).then(() => {
        this.chatStore.unsetUser();
        bus.emit(LOGGED_OUT, null);
      }).finally(()=>{
          this.isLoggingOut = false
          this.chatStore.decrementProgressCount();
      });
    },
    shouldDisplayLogout() {
      return this.chatStore.currentUser != null;
    },
    onNotificationsClicked() {
      bus.emit(OPEN_NOTIFICATIONS_DIALOG);
    },
    openFiles() {
      bus.emit(OPEN_VIEW_FILES_DIALOG, { chatId: this.$route.params.id });
    },
    openPinnedMessages() {
      bus.emit(OPEN_PINNED_MESSAGES_MODAL, {chatId: this.$route.params.id});
    },
    openPublishedMessages() {
        bus.emit(OPEN_PUBLISHED_MESSAGES_MODAL, {chatId: this.$route.params.id});
    },
    shouldPinnedMessages() {
      return this.chatStore.currentUser && this.isChatable();
    },
    shouldPublishedMessages() {
      return this.chatStore.currentUser && this.isChatable();
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
