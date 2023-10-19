<template>
  <v-app>

      <v-app-bar
          color='indigo'
          id="myAppBar"
          :density="getDensity()"
      >
          <v-progress-linear
            v-if="showProgress"
            indeterminate
            color="white"
            absolute
          ></v-progress-linear>

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
                       :content="chatStore.videoChatUsersCount"
                       :model-value="!!chatStore.videoChatUsersCount"
                       color="green"
                       overlap
                       offset-y="1.8em"
              >
                  <v-btn v-if="chatStore.showCallButton" icon @click="createCall()" :title="chatStore.tetATet ? $vuetify.locale.t('$vuetify.call_up') : $vuetify.locale.t('$vuetify.enter_into_call')">
                      <v-icon :x-large="isMobile()" color="green">{{chatStore.tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                  </v-btn>
                  <v-btn v-if="chatStore.showHangButton" icon @click="stopCall()" :title="$vuetify.locale.t('$vuetify.leave_call')">
                      <v-icon :x-large="isMobile()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'text-red'">mdi-phone</v-icon>
                  </v-btn>
              </v-badge>
          </template>

          <v-btn v-if="chatStore.showGoToBlogButton && !isMobile()" icon :href="goToBlogLink()" :title="$vuetify.locale.t('$vuetify.go_to_blog_post')">
            <v-icon>mdi-postage-stamp</v-icon>
          </v-btn>

          <v-btn v-if="chatStore.showScrollDown" icon @click="scrollDown()" :title="$vuetify.locale.t('$vuetify.scroll_down')">
            <v-icon :x-large="isMobile()">mdi-arrow-down-thick</v-icon>
          </v-btn>
          <v-btn v-if="shouldShowFileUpload" icon @click="onShowFileUploadClicked()" :title="$vuetify.locale.t('$vuetify.show_upload_files')">
            <v-icon :x-large="isMobile()">mdi-cloud-upload</v-icon>
          </v-btn>

          <img v-if="!!chatStore.avatar && !isMobile()" @click="onChatAvatarClick()" class="ml-2 v-avatar chat-avatar" :src="chatStore.avatar"/>
          <div class="d-flex flex-column app-title mx-2" :class="isInChat() ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked()" :style="{'cursor': isInChat() ? 'pointer' : 'default'}">
            <div :class="!isMobile() ? ['align-self-center'] : []" class="app-title-text" v-html="chatStore.title"></div>
            <div v-if="shouldShowSubtitle()" :class="!isMobile() ? ['align-self-center'] : []" class="app-title-subtext">
              {{ getSubtitle() }}
            </div>
          </div>

          <v-card variant="plain" min-width="330" v-if="chatStore.isShowSearch" style="margin-left: 1.2em; margin-right: 2px">
            <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line v-model="searchStringFacade" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" :label="searchName()">
              <template v-slot:append-inner>
                <v-btn icon density="compact" @click.prevent="switchSearchType()" :disabled="!canSwitchSearchType()"><v-icon class="search-icon">{{ searchIcon }}</v-icon></v-btn>
              </template>
            </v-text-field>
          </v-card>

      </v-app-bar>

      <v-main>
        <v-container fluid class="ma-0 pa-0" style="height: 100%; width: 100%">

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

          <router-view />
        </v-container>

        <!-- We store modals outside of container in order they not to contribute into the height -->
        <LoginModal/>
        <SettingsModal/>
        <SimpleModal/>
        <FileUploadModal/>
        <FileListModal/>
        <FileItemAttachToMessage/>
        <NotificationsModal/>
        <MessageReadUsersModal/>
        <PinnedMessagesModal/>
        <ChatParticipantsModal/>
        <MessageResendToModal/>
        <FileTextEditModal/>
        <PlayerModal/>
        <ChatEditModal/>
    </v-main>

    <v-navigation-drawer location="right" v-model="chatStore.showDrawer">
        <RightPanelActions/>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import {getBlogLink, hasLength, isChatRoute} from "@/utils";
import {
    chat_list_name,
    chat_name,
    confirmation_pending_name,
    forgot_password_name, password_restore_check_email_name, password_restore_enter_new_name,
    registration_name,
    videochat_name
} from "@/router/routes";
import axios from "axios";
import bus, {
  CHAT_ADD,
  CHAT_DELETED,
  CHAT_EDITED, FOCUS,
  LOGGED_OUT, NOTIFICATION_ADD, NOTIFICATION_DELETE, OPEN_FILE_UPLOAD_MODAL, OPEN_PARTICIPANTS_DIALOG, PLAYER_MODAL,
  PROFILE_SET,
  SCROLL_DOWN, UNREAD_MESSAGES_CHANGED, USER_PROFILE_CHANGED, VIDEO_CALL_INVITED, VIDEO_CALL_SCREEN_SHARE_CHANGED,
  VIDEO_CALL_USER_COUNT_CHANGED, VIDEO_DIAL_STATUS_CHANGED, VIDEO_RECORDING_CHANGED,
} from "@/bus/bus";
import LoginModal from "@/LoginModal.vue";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'
import {
  searchStringFacade,
  SEARCH_MODE_CHATS,
  SEARCH_MODE_MESSAGES,
  SEARCH_MODE_USERS,
  goToPreserving
} from "@/mixins/searchString";
import RightPanelActions from "@/RightPanelActions.vue";
import SettingsModal from "@/SettingsModal.vue";
import SimpleModal from "@/SimpleModal.vue";
import FileListModal from "@/FileListModal.vue";
import {createGraphQlClient, destroyGraphqlClient} from "@/graphql/graphql";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import FileUploadModal from "@/FileUploadModal.vue";
import FileItemAttachToMessage from "@/FileItemAttachToMessage.vue";
import NotificationsModal from "@/NotificationsModal.vue";
import MessageReadUsersModal from "@/MessageReadUsersModal.vue"
import PinnedMessagesModal from "@/PinnedMessagesModal.vue";
import ChatParticipantsModal from "@/ChatParticipantsModal.vue";
import MessageResendToModal from "@/MessageResendToModal.vue";
import FileTextEditModal from "@/FileTextEditModal.vue";
import PlayerModal from "@/PlayerModal.vue";
import ChatEditModal from "@/ChatEditModal.vue";
import {once} from "lodash/function";

const getGlobalEventsData = (message) => {
  return message.data?.globalEvents
};

export default {
    mixins: [
        searchStringFacade(),
        graphqlSubscriptionMixin('globalEvents'),
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
        showProgress() {
            return this.chatStore.progressCount > 0
        },
        currentUserAvatar() {
            return this.chatStore.currentUser?.avatar;
        },
        // it differs from original
        chatId() {
          if (!this.isInChat()) {
            return null
          } else {
            return this.$route.params.id
          }
        },
        notificationsCount() {
            return this.chatStore.notificationsCount
        },
        showNotificationBadge() {
            return this.notificationsCount != 0 && !this.chatStore.showDrawer
        },
        searchIcon() {
          if (this.chatStore.searchType == SEARCH_MODE_CHATS) {
            return 'mdi-forum'
          } else if (this.chatStore.searchType == SEARCH_MODE_MESSAGES) {
            return 'mdi-message-text-outline'
          } else if (this.chatStore.searchType == SEARCH_MODE_USERS) {
            return 'mdi-account-group'
          }
        },
        shouldShowFileUpload() {
            return !!this.chatStore.fileUploadingQueue.length
        },
    },
    methods: {
        getStore() {
            return this.chatStore
        },
        resetInput() {
          this.searchStringFacade = null
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
            this.chatStore.showDrawer = !this.chatStore.showDrawer;
        },
        createCall() {
            console.debug("createCall");
            axios.put(`/api/video/${this.chatId}/dial/start`).then(()=>{
                const routerNewState = { name: videochat_name};
                goToPreserving(this.$route, this.$router, routerNewState);
                this.updateLastAnsweredTimestamp();
            })
        },
        stopCall() {
            console.debug("stopping Call");
            this.chatStore.leavingVideoAcceptableParam = true;
            const routerNewState = { name: chat_name };
            goToPreserving(this.$route, this.$router, routerNewState);
            this.updateLastAnsweredTimestamp();
        },
        updateLastAnsweredTimestamp() {
            this.lastAnswered = +new Date();
        },

        onProfileSet(){
            this.chatStore.fetchNotificationsCount();
            this.graphQlSubscribe();
        },
        onLoggedOut() {
            this.resetVariables();
            this.graphQlUnsubscribe();
        },
        resetVariables() {
            this.chatStore.unsetNotifications();
        },
        canSwitchSearchType() {
            return this.isInChat() || this.$route.name == chat_list_name
        },
        switchSearchType() {
          this.chatStore.switchSearchType()
        },
        scrollDown () {
          bus.emit(SCROLL_DOWN)
        },
        goToBlogLink() {
          return getBlogLink(this.chatId)
        },
        searchName() {
            if (this.chatStore.searchType == SEARCH_MODE_CHATS) {
              return this.$vuetify.locale.t('$vuetify.search_in_chats')
            } else if (this.chatStore.searchType == SEARCH_MODE_MESSAGES) {
              return this.$vuetify.locale.t('$vuetify.search_in_messages')
            } else if (this.chatStore.searchType == SEARCH_MODE_USERS) {
              return this.$vuetify.locale.t('$vuetify.find_user')
            }
        },
        getGraphQlSubscriptionQuery() {
          return `
                  subscription {
                    globalEvents {
                      eventType
                      chatEvent {
                        id
                        name
                        avatar
                        avatarBig
                        shortInfo
                        lastUpdateDateTime
                        participantIds
                        canEdit
                        canDelete
                        canLeave
                        unreadMessages
                        canBroadcast
                        canVideoKick
                        canChangeChatAdmins
                        tetATet
                        canAudioMute
                        participantsCount
                        participants {
                          id
                          login
                          avatar
                          admin
                          shortInfo
                        }
                        canResend
                        pinned
                        blog
                      }
                      chatDeletedEvent {
                        id
                      }
                      userEvent {
                        id
                        login
                        avatar
                      }
                      videoUserCountChangedEvent {
                        usersCount
                        chatId
                      }
                      videoCallScreenShareChangedDto {
                        chatId
                        hasScreenShares
                      }
                      videoRecordingChangedEvent {
                        recordInProgress
                        chatId
                      }
                      videoCallInvitation {
                        chatId
                        chatName
                      }
                      videoParticipantDialEvent {
                        chatId
                        dials {
                          userId
                          status
                        }
                      }
                      unreadMessagesNotification {
                        chatId
                        unreadMessages
                        lastUpdateDateTime
                      }
                      allUnreadMessagesNotification {
                        allUnreadMessages
                      }
                      notificationEvent {
                        id
                        chatId
                        messageId
                        notificationType
                        description
                        createDateTime
                        byUserId
                        byLogin
                        chatTitle
                      }
                    }
                  }
              `
        },
        onNextSubscriptionElement(e) {
          if (getGlobalEventsData(e).eventType === 'chat_created') {
            const d = getGlobalEventsData(e).chatEvent;
            bus.emit(CHAT_ADD, d);
          } else if (getGlobalEventsData(e).eventType === 'chat_edited') {
            const d = getGlobalEventsData(e).chatEvent;
            bus.emit(CHAT_EDITED, d);
          } else if (getGlobalEventsData(e).eventType === 'chat_deleted') {
            const d = getGlobalEventsData(e).chatDeletedEvent;
            bus.emit(CHAT_DELETED, d);
          } else if (getGlobalEventsData(e).eventType === 'user_profile_changed') {
            const d = getGlobalEventsData(e).userEvent;
            bus.emit(USER_PROFILE_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === "video_user_count_changed") {
            const d = getGlobalEventsData(e).videoUserCountChangedEvent;
            bus.emit(VIDEO_CALL_USER_COUNT_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === "video_screenshare_changed") {
            const d = getGlobalEventsData(e).videoCallScreenShareChangedDto;
            bus.emit(VIDEO_CALL_SCREEN_SHARE_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === "video_recording_changed") {
            const d = getGlobalEventsData(e).videoRecordingChangedEvent;
            bus.emit(VIDEO_RECORDING_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === 'video_call_invitation') {
            const d = getGlobalEventsData(e).videoCallInvitation;
            bus.emit(VIDEO_CALL_INVITED, d);
          } else if (getGlobalEventsData(e).eventType === "video_dial_status_changed") {
            const d = getGlobalEventsData(e).videoParticipantDialEvent;
            bus.emit(VIDEO_DIAL_STATUS_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === 'chat_unread_messages_changed') {
            const d = getGlobalEventsData(e).unreadMessagesNotification;
            bus.emit(UNREAD_MESSAGES_CHANGED, d);
          } else if (getGlobalEventsData(e).eventType === 'notification_add') {
            const d = getGlobalEventsData(e).notificationEvent;
            bus.emit(NOTIFICATION_ADD, d);
          } else if (getGlobalEventsData(e).eventType === 'notification_delete') {
            const d = getGlobalEventsData(e).notificationEvent;
            bus.emit(NOTIFICATION_DELETE, d);
          }
        },
        onChatAvatarClick() {
          bus.emit(PLAYER_MODAL, {canShowAsImage: true, url: this.chatStore.avatar})
        },
        isInChat() {
          return isChatRoute(this.$route)
        },
        onInfoClicked() {
          if (this.isInChat()) {
            bus.emit(OPEN_PARTICIPANTS_DIALOG, this.chatId);
          }
        },
        onShowFileUploadClicked() {
            bus.emit(OPEN_FILE_UPLOAD_MODAL, { });
        },
        onFocus(e) {
            // console.log("Focus", e);
            if (this.chatStore.currentUser) {
                this.chatStore.fetchNotificationsCount();
            }
            bus.emit(FOCUS);
        },
        getSubtitle() {
            if (!!this.chatStore.moreImportantSubtitleInfo) {
                return this.chatStore.moreImportantSubtitleInfo
            } else {
                return this.chatStore.chatUsersCount + " " + this.$vuetify.locale.t('$vuetify.participants')
            }
        },
        shouldShowSubtitle() {
            return !!this.chatStore.chatUsersCount || !!this.chatStore.moreImportantSubtitleInfo
        },
        afterRouteInitialized() {
            this.chatStore.fetchAvailableOauth2Providers().then(() => {
                if (this.$route.name == registration_name || this.$route.name == confirmation_pending_name || this.$route.name == forgot_password_name || this.$route.name == password_restore_enter_new_name || this.$route.name == password_restore_check_email_name || this.$route.name == confirmation_pending_name) {
                    return
                }
                this.chatStore.fetchUserProfile();
            })
        },
    },
    components: {
        ChatEditModal,
        RightPanelActions,
        LoginModal,
        SettingsModal,
        SimpleModal,
        FileUploadModal,
        FileListModal,
        FileItemAttachToMessage,
        NotificationsModal,
        MessageReadUsersModal,
        PinnedMessagesModal,
        ChatParticipantsModal,
        MessageResendToModal,
        FileTextEditModal,
        PlayerModal,
    },
    created() {
        createGraphQlClient();

        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);

        addEventListener("focus", this.onFocus);

        this.afterRouteInitialized = once(this.afterRouteInitialized);

        this.$router.afterEach((to, from) => {
            this.afterRouteInitialized()
        })
    },
    beforeUnmount() {
        removeEventListener("focus", this.onFocus);

        this.graphQlUnsubscribe();
        destroyGraphqlClient();

        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
    },

    watch: {
      'chatStore.currentUser': function(newUserValue, oldUserValue) {
        console.debug("User new", newUserValue, "old" , oldUserValue);
        if (newUserValue && !oldUserValue) {
            bus.emit(PROFILE_SET);
        }
      },
    }
}
</script>

<style lang="scss">
@use './styles/settings';

.search-icon {
  opacity: settings.$list-item-icon-opacity;
}

.app-title {
  width: 100%;

  &-text {
    font-size: .875rem;
    font-weight: 500;
    letter-spacing: .09em;
    height: 1.6em;
    white-space: break-spaces;
    overflow: hidden;
  }

  &-subtext {
    font-size: .7rem;
    letter-spacing: initial;
    text-transform: initial;
    opacity: 50%;
    height: 1.6em;
    white-space: break-spaces;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  &-hoverable {
    color: white
  }

  &-hoverable:hover {
    background-color: #4e5fbb;
    border-radius: 4px;
  }
}

.chat-avatar {
  display: block;
  max-width: 36px;
  max-height: 36px;
  width: auto;
  height: auto;
  cursor: pointer
}

.v-card {
  .v-pagination__list {
    justify-content: start;
  }
}

</style>

<style lang="stylus">
@import "constants.styl"

.colored-link {
    color: $linkColor;
    text-decoration none
}

.list-item-prepend-spacer-16 {
    .v-list-item__prepend {
        .v-list-item__spacer {
            width: 16px
        }
    }
}

</style>
