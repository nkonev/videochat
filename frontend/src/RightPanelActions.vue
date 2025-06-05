<template>
    <v-list>
      <v-list-item id="right-panel-user-login" v-if="chatStore.currentUser" @click="onUserClick">
          <template v-slot:prepend v-if="hasLength(chatStore.currentUser.avatar)">
              <v-badge
                  :color="getUserBadgeColor(userState)"
                  dot
                  location="right bottom"
                  overlap
                  bordered
                  :model-value="userState.online"
              >
                  <v-avatar :image="chatStore.currentUser.avatar"></v-avatar>
              </v-badge>
          </template>
          <template v-slot:title>
              <span :style="getLoginColoredStyle(chatStore.currentUser)" v-html="getUserNameOverride(chatStore.currentUser, userState)"></span>
          </template>
          <template v-slot:subtitle>
              <span v-html="chatStore.currentUser.shortInfo"></span>
          </template>
      </v-list-item>
    </v-list>

    <v-divider></v-divider>

    <v-list density="compact" nav>
      <v-list-item v-if="shouldShowHome()" @click.prevent="goHome()" :href="getRouteRoot()" prepend-icon="mdi-home" :title="$vuetify.locale.t('$vuetify.start')"></v-list-item>
      <v-list-item v-if="shouldShowUpperChats()" @click.prevent="goChats()" :href="getRouteChats()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
      <v-list-item v-if="shouldDisplayCopyCallLink()" @click.prevent="copyCallLink()" prepend-icon="mdi-content-copy" :title="$vuetify.locale.t('$vuetify.copy_video_call_link')"></v-list-item>
      <v-list-item v-if="shouldShowLowerChats()" @click.prevent="goChats()" :href="getRouteChats()" prepend-icon="mdi-forum" :title="$vuetify.locale.t('$vuetify.chats')"></v-list-item>
      <v-list-item v-if="shouldShowGoToBlogPost()" @click.prevent="goToBlogPost()" :href="getRouteGoToBlogPost()" prepend-icon="mdi-postage-stamp" :title="$vuetify.locale.t('$vuetify.go_to_blog_post')"></v-list-item>
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
      <v-list-item v-if="shouldShowAdminsCorner()" @click.prevent="openAdminsCorner()" :href="getRouteAdminsCorner()" prepend-icon="mdi-tools" :title="$vuetify.locale.t('$vuetify.admins_corner')"></v-list-item>
      <v-list-item :disabled="isLoggingOut" @click.prevent="logout()" v-if="shouldDisplayLogout()" prepend-icon="mdi-logout" :title="$vuetify.locale.t('$vuetify.logout')"></v-list-item>
    </v-list>

</template>

<script>
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {
    admins_corner, admins_corner_name,
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
  LOGGED_OUT, OPEN_CHAT_EDIT,
  OPEN_NOTIFICATIONS_DIALOG,
  OPEN_PINNED_MESSAGES_MODAL, OPEN_PUBLISHED_MESSAGES_MODAL,
  OPEN_SETTINGS,
  OPEN_VIEW_FILES_DIALOG,
  WEBSOCKET_INITIALIZED, WEBSOCKET_UNINITIALIZED,
} from "@/bus/bus";
import {copyCallLink, getBlogLink, getLoginColoredStyle, hasLength, isChatRoute} from "@/utils";
import userStatusMixin from "@/mixins/userStatusMixin.js";
import onFocusMixin from "@/mixins/onFocusMixin.js";

const userStateFactory = () => {
    return {
        isInVideo: false,
        online: false,
    }
}

export default {
  mixins: [
      userStatusMixin('userStatusInUserList'), // subscription
      onFocusMixin(),
  ],
  data() {
      return {
          isLoggingOut: false,
          userState: userStateFactory(),
          initialized: false,
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
    hasLength,
    getLoginColoredStyle,
    getUserNameOverride(currentUser, userState) {
      const item = {
        login: currentUser.login,
        avatar: currentUser.avatar,
        online: userState.online,
        isInVideo: userState.isInVideo
      };
      return this.getUserName(item)
    },
    getUserIdsSubscribeTo() {
      return [this.chatStore.currentUser?.id];
    },
    onUserStatusChanged(dtos) {
      const userId = this.chatStore.currentUser?.id;
      if (dtos && userId) {
        dtos.forEach(dtoItem => {
          if (dtoItem.online !== null && userId == dtoItem.userId) {
            this.userState.online = dtoItem.online;
          }
          if (dtoItem.isInVideo !== null && userId == dtoItem.userId) {
            this.userState.isInVideo = dtoItem.isInVideo;
          }
        })
      }
    },

    createChat() {
      bus.emit(OPEN_CHAT_EDIT, null);
    },
    editChat() {
      bus.emit(OPEN_CHAT_EDIT, this.chatId);
    },
    shouldDisplayCopyCallLink() {
      return (this.chatStore.showCallManagement) && this.isMobile()
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
      this.$router.push({name: root_name})
    },
    copyCallLink() {
      copyCallLink(this.chatId);
      this.setTempNotification(this.$vuetify.locale.t('$vuetify.video_call_link_copied'));
    },
    getRouteChats() {
      return chats
    },
    getRouteBlogs() {
      return blog
    },
    getRouteGoToBlogPost() {
      return getBlogLink(this.chatId)
    },
    getRouteUsers() {
      return profiles;
    },
    goChats() {
      // we don't store query here because
      // user on mobile searches by messages and typed "ololo"
      // then user clicks Chats
      // then user enter into the different chat
      // due to "ololo" the user will see 0 messages
      this.$router.push({name: chat_list_name})
    },
    goBlogs() {
      window.location.href = blog
    },
    goToBlogPost() {
      window.location.href = this.getRouteGoToBlogPost()
    },
    openUsers() {
      this.chatStore.incrementProgressCount();
      this.$router.push({name: profile_list_name}).finally(() => {
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
    logout() {
      this.isLoggingOut = true;
      this.chatStore.incrementProgressCount();
      axios.post(`/api/aaa/logout`).then(() => {
        this.chatStore.unsetUser();
        bus.emit(LOGGED_OUT);
      }).finally(() => {
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
      bus.emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.$route.params.id});
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
      return true
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
    shouldShowGoToBlogPost() {
      if (!this.isMobile()) {
        return false
      } else {
        return this.chatStore.showGoToBlogButton
      }
    },
    onProfileSet() {
      this.graphQlUserStatusSubscribe();
    },
    doInitialize() {
      if (!this.initialized) {
        this.initialized = true;
        this.onProfileSet();
      }
    },
    onLogOut() {
      this.userState = userStateFactory();
      this.graphQlUserStatusUnsubscribe();
    },
    doUninitialize() {
      if (this.initialized) {
        this.onLogOut();
        this.initialized = false;
      }
    },
    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
    onUserClick() {
      bus.emit(OPEN_SETTINGS, 'user_profile_self')
    },
    shouldShowAdminsCorner() {
      return this.chatStore.currentUser?.canShowAdminsCorner
    },
    openAdminsCorner() {
      this.$router.push({name: admins_corner_name})
    },
    getRouteAdminsCorner() {
      return admins_corner
    },
    requestStatuses() {
      this.$nextTick(() => {
        if (this.chatStore.currentUser) {
          const userIds = this.chatStore.currentUser.id;
          this.triggerUsesStatusesEvents(userIds, this.requestAbortController.signal);
        }
      })
    },
    onFocus() {
      this.requestStatuses();
    }
  },
  mounted() {
      if (this.canDrawUsers()) {
          this.doInitialize();
      }

      bus.on(WEBSOCKET_INITIALIZED, this.doInitialize);
      bus.on(WEBSOCKET_UNINITIALIZED, this.doUninitialize);
      this.installOnFocus();
  },
  beforeUnmount() {
      this.doUninitialize();

      this.uninstallOnFocus();
      bus.off(WEBSOCKET_INITIALIZED, this.doInitialize);
      bus.off(WEBSOCKET_UNINITIALIZED, this.doUninitialize);

  },

}
</script>

<style lang="scss">
@use './styles/settings';

.notifications-badge {

    .notification-icon {
        opacity: settings.$list-item-icon-opacity;
    }
}

#right-panel-user-login .v-list-item__prepend > .v-badge ~ .v-list-item__spacer {
    width: 16px;
}
</style>
