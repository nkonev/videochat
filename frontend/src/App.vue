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

          <template v-if="chatStore.showCallManagement">
            <template v-if="showSearchButton || !isMobile()">
                <v-badge
                         :content="chatStore.videoChatUsersCount"
                         :model-value="!!chatStore.videoChatUsersCount"
                         color="green"
                         overlap
                         offset-y="1.8em"
                >
                    <v-btn v-if="chatStore.isReady()" icon :loading="chatStore.initializingVideoCall" @click="createCall()" :title="chatStore.tetATet ? $vuetify.locale.t('$vuetify.call_up') : $vuetify.locale.t('$vuetify.enter_into_call')">
                        <v-icon :size="getIconSize()" color="green">{{chatStore.tetATet ? 'mdi-phone' : 'mdi-phone-plus'}}</v-icon>
                    </v-btn>
                    <v-btn v-else-if="chatStore.isInCall()" icon :loading="chatStore.initializingVideoCall" @click="stopCall()" :title="$vuetify.locale.t('$vuetify.leave_call')">
                        <v-icon :size="getIconSize()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'text-red'">mdi-phone</v-icon>
                    </v-btn>
                </v-badge>
            </template>

            <template v-if="!isMobile()">
              <template v-if="chatStore.isInCall()">
                <template v-if="chatStore.canShowMicrophoneButton">
                  <v-btn v-if="chatStore.showMicrophoneOnButton" icon @click="offMicrophone()" :title="$vuetify.locale.t('$vuetify.mute_audio')"><v-icon>mdi-microphone</v-icon></v-btn>
                  <v-btn v-if="chatStore.showMicrophoneOffButton" icon @click="onMicrophone()" :title="$vuetify.locale.t('$vuetify.unmute_audio')"><v-icon>mdi-microphone-off</v-icon></v-btn>
                </template>

                <v-btn icon @click="addScreenSource()" :title="$vuetify.locale.t('$vuetify.screen_share')">
                  <v-icon>mdi-monitor-screenshot</v-icon>
                </v-btn>
                <v-btn icon @click="addVideoSource()" :title="$vuetify.locale.t('$vuetify.source_add')">
                  <v-icon>mdi-video-plus</v-icon>
                </v-btn>
              </template>

              <v-btn v-if="chatStore.showRecordStartButton" icon @click="startRecord()" :loading="chatStore.initializingStaringVideoRecord" :title="$vuetify.locale.t('$vuetify.start_record')">
                <v-icon>mdi-record-rec</v-icon>
              </v-btn>
              <v-btn v-if="chatStore.showRecordStopButton" icon @click="stopRecord()" :loading="chatStore.initializingStoppingVideoRecord" :title="$vuetify.locale.t('$vuetify.stop_record')">
                <v-icon color="red">mdi-stop</v-icon>
              </v-btn>
            </template>
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

        <template v-if="showSearchButton">
          <img v-if="!!chatStore.avatar && !isMobile()" @click="onChatAvatarClick()" class="ml-2 v-avatar chat-avatar" :src="chatStore.avatar"/>
          <div class="d-flex flex-column app-title mx-2" :class="isInChat() ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked()" :style="{'cursor': isInChat() ? 'pointer' : 'default'}">
            <div :class="!isMobile() ? ['align-self-center'] : []" class="app-title-text" v-html="chatStore.title"></div>
            <div v-if="shouldShowSubtitle()" :class="!isMobile() ? ['align-self-center'] : []" class="app-title-subtext">
              {{ getSubtitle() }}
            </div>
          </div>
        </template>

        <template v-if="chatStore.isShowSearch">
          <CollapsedSearch :provider="{
              getModelValue: this.getModelValue,
              setModelValue: this.setModelValue,
              getShowSearchButton: this.getShowSearchButton,
              setShowSearchButton: this.setShowSearchButton,
              searchName: this.searchName,
              switchSearchType: this.switchSearchType,
              canSwitchSearchType: this.canSwitchSearchType,
              searchIcon: this.searchIcon,
              textFieldVariant: 'solo',
          }"/>
        </template>
      </v-app-bar>

      <v-main>
        <v-container fluid class="ma-0 pa-0" style="height: 100%; width: 100%">

          <v-snackbar v-model="chatStore.showAlert" :color="chatStore.errorColor" timeout="-1" :multi-line="true" :transition="false">
            {{ chatStore.lastError }}

            <template v-slot:actions>
              <v-btn
                text
                v-if="chatStore.errorColor == 'error'"
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
          <v-snackbar v-model="showWebsocketRestored" color="black" timeout="-1" :multi-line="true" :transition="false">
            {{ $vuetify.locale.t('$vuetify.websocket_restored') }}
            <template v-slot:actions>
              <v-btn
                text
                @click="onPressWebsocketRestored()"
              >
                {{ $vuetify.locale.t('$vuetify.btn_update') }}
              </v-btn>
              <v-btn text @click="showWebsocketRestored = false">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>

            </template>
          </v-snackbar>
          <v-snackbar v-model="invitedVideoChatAlert" color="success" timeout="-1" :multi-line="true" :transition="false">
            <span class="call-blink">
                {{ $vuetify.locale.t('$vuetify.you_called', invitedVideoChatId, invitedVideoChatName) }}
            </span>
            <template v-slot:actions>
              <v-btn icon size="x-large" @click="onClickInvitation()"><v-icon size="x-large" color="white">mdi-phone</v-icon></v-btn>
              <v-btn icon size="x-large" @click="onClickCancelInvitation()"><v-icon size="x-large" color="white">mdi-close-circle</v-icon></v-btn>
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
        <PermissionsWarningModal/>
        <VideoAddNewSourceModal/>
        <MessageEditModal/>
    </v-main>

    <v-navigation-drawer :location="isMobile() ? 'left' : 'right'" v-model="chatStore.showDrawer">
        <RightPanelActions/>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import {getBlogLink, hasLength, isCalling, isChatRoute} from "@/utils";
import {
    chat_list_name,
    chat_name,
    confirmation_pending_name,
    forgot_password_name, check_email_name, password_restore_enter_new_name,
    registration_name,
    videochat_name, registration_resend_email_name
} from "@/router/routes";
import axios from "axios";
import bus, {
  ADD_SCREEN_SOURCE, ADD_VIDEO_SOURCE_DIALOG,
  CHAT_ADD,
  CHAT_DELETED,
  CHAT_EDITED, CHAT_REDRAW,
  FOCUS,
  LOGGED_OUT,
  NOTIFICATION_ADD,
  NOTIFICATION_DELETE,
  OPEN_FILE_UPLOAD_MODAL,
  OPEN_PARTICIPANTS_DIALOG,
  OPEN_PERMISSIONS_WARNING_MODAL,
  PLAYER_MODAL,
  PROFILE_SET,
  REFRESH_ON_WEBSOCKET_RESTORED,
  SCROLL_DOWN, SET_LOCAL_MICROPHONE_MUTED,
  UNREAD_MESSAGES_CHANGED,
  PARTICIPANT_CHANGED,
  VIDEO_CALL_INVITED,
  VIDEO_CALL_SCREEN_SHARE_CHANGED,
  VIDEO_CALL_USER_COUNT_CHANGED,
  VIDEO_DIAL_STATUS_CHANGED,
  VIDEO_RECORDING_CHANGED,
  WEBSOCKET_RESTORED,
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
import PermissionsWarningModal from "@/PermissionsWarningModal.vue";
import {prefix} from "@/router/routes"
import VideoAddNewSourceModal from "@/VideoAddNewSourceModal.vue";
import MessageEdit from "@/MessageEdit.vue";
import MessageEditModal from "@/MessageEditModal.vue";
import CollapsedSearch from "@/CollapsedSearch.vue";

const audio = new Audio(`${prefix}/call.mp3`);

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
            showSearchButton: true,
            showWebsocketRestored: false,
            invitedVideoChatId: 0,
            invitedVideoChatName: null,
            invitedVideoChatAlert: false,
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
        shouldShowFileUpload() {
            return !!this.chatStore.fileUploadingQueue.length
        },
    },
    methods: {
        searchIcon() {
            if (this.chatStore.searchType == SEARCH_MODE_CHATS) {
                return 'mdi-forum'
            } else if (this.chatStore.searchType == SEARCH_MODE_MESSAGES) {
                return 'mdi-message-text-outline'
            } else if (this.chatStore.searchType == SEARCH_MODE_USERS) {
                return 'mdi-account-group'
            }
        },
        getStore() {
            return this.chatStore
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
            const routerNewState = { name: videochat_name};
            goToPreserving(this.$route, this.$router, routerNewState);
        },
        stopCall() {
            console.debug("stopping Call");
            this.chatStore.leavingVideoAcceptableParam = true;
            const routerNewState = { name: chat_name };
            goToPreserving(this.$route, this.$router, routerNewState);
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
            this.resetVideoInvitation()
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
              return this.$vuetify.locale.t('$vuetify.search_by_users')
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
                        availableToSearch
                        isResultFromSearch
                        pinned
                        blog
                      }
                      chatDeletedEvent {
                        id
                      }
                      participantEvent {
                        id
                        login
                        avatar
                        shortInfo
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
                        status
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
          } else if (getGlobalEventsData(e).eventType === 'chat_redraw') {
            const d = getGlobalEventsData(e).chatEvent;
            bus.emit(CHAT_REDRAW, d);
          } else if (getGlobalEventsData(e).eventType === 'chat_deleted') {
            const d = getGlobalEventsData(e).chatDeletedEvent;
            bus.emit(CHAT_DELETED, d);
          } else if (getGlobalEventsData(e).eventType === 'participant_changed') {
            const d = getGlobalEventsData(e).participantEvent;
            bus.emit(PARTICIPANT_CHANGED, d);
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
            return this.chatStore.fetchAvailableOauth2Providers()
        },
        fetchProfileIfNeed() {
            if (!this.chatStore.currentUser) {
                if (this.$route.name == registration_name || this.$route.name == confirmation_pending_name || this.$route.name == forgot_password_name || this.$route.name == password_restore_enter_new_name || this.$route.name == check_email_name || this.$route.name == confirmation_pending_name || this.$route.name == registration_resend_email_name) {
                    return
                }
                this.chatStore.fetchUserProfile();
            }
        },
        onWsRestored() {
          this.showWebsocketRestored = true;
        },
        onPressWebsocketRestored() {
          this.showWebsocketRestored = false;
          bus.emit(REFRESH_ON_WEBSOCKET_RESTORED);
        },
        resetVideoInvitation() {
          this.invitedVideoChatAlert = false;
          this.invitedVideoChatId = 0;
          this.invitedVideoChatName = null;
        },
        onVideoCallInvited(data) {
          if (isCalling(data.status)) {
            this.invitedVideoChatId = data.chatId;
            this.invitedVideoChatName = data.chatName;
            this.invitedVideoChatAlert = true;

            audio.play().catch(error => {
              console.warn("Unable to play sound", error);
              bus.emit(OPEN_PERMISSIONS_WARNING_MODAL);
            })
          } else {
            this.resetVideoInvitation();
          }
        },
        onClickInvitation() {
          const routerNewState = { name: videochat_name, params: { id: this.invitedVideoChatId }};
          goToPreserving(this.$route, this.$router, routerNewState)
          this.resetVideoInvitation();
        },
        onClickCancelInvitation() {
          axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`).then(()=>{
            this.resetVideoInvitation();
          });
        },
        onMicrophone() {
          bus.emit(SET_LOCAL_MICROPHONE_MUTED, false);
        },
        offMicrophone() {
          bus.emit(SET_LOCAL_MICROPHONE_MUTED, true);
        },
        addVideoSource() {
          bus.emit(ADD_VIDEO_SOURCE_DIALOG);
        },
        addScreenSource() {
          bus.emit(ADD_SCREEN_SOURCE);
        },
        onVideRecordingChanged(e) {
          if (this.isVideoRoute()) {
            this.chatStore.showRecordStartButton = !e.recordInProgress;
            this.chatStore.showRecordStopButton = e.recordInProgress;
          } else if (e.recordInProcess) {
            this.chatStore.showRecordStartButton = !e.recordInProgress;
            this.chatStore.showRecordStopButton = e.recordInProgress;
          }
          if (this.chatStore.initializingStaringVideoRecord && e.recordInProgress) {
            this.chatStore.initializingStaringVideoRecord = false;
          }
          if (this.chatStore.initializingStoppingVideoRecord && !e.recordInProgress) {
            this.chatStore.initializingStoppingVideoRecord = false;
          }
        },
        startRecord() {
          axios.put(`/api/video/${this.chatId}/record/start`);
          this.chatStore.initializingStaringVideoRecord = true;
        },
        stopRecord() {
          axios.put(`/api/video/${this.chatId}/record/stop`);
          this.chatStore.initializingStoppingVideoRecord = true;
        },
        isVideoRoute() {
          return this.$route.name == videochat_name
        },
        getIconSize() {
            if (this.isMobile()) {
              return 'x-large'
            } else {
              return undefined
            }
        },
        getModelValue() {
            return this.searchStringFacade
        },
        setModelValue(v) {
            this.searchStringFacade = v
        },
        getShowSearchButton() {
            return this.showSearchButton
        },
        setShowSearchButton(v) {
            this.showSearchButton = v
        },
    },
    components: {
      MessageEdit,
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
        PermissionsWarningModal,
        VideoAddNewSourceModal,
        MessageEditModal,
        CollapsedSearch,
    },
    created() {
        createGraphQlClient();

        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);
        bus.on(WEBSOCKET_RESTORED, this.onWsRestored);
        bus.on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
        bus.on(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);

        addEventListener("focus", this.onFocus);

        // It's placed after each route in order not to have a race-condition
        this.afterRouteInitialized = once(this.afterRouteInitialized);
        this.$router.afterEach((to, from) => {
            this.afterRouteInitialized().then(()=>{
                this.fetchProfileIfNeed();
            })
        })
    },
    beforeUnmount() {
        removeEventListener("focus", this.onFocus);

        this.graphQlUnsubscribe();
        destroyGraphqlClient();

        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
        bus.off(WEBSOCKET_RESTORED, this.onWsRestored);
        bus.off(VIDEO_CALL_INVITED, this.onVideoCallInvited);
        bus.off(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
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

// removes extraneous scroll at right side of the screen on Chrome
html {
  overflow-y: unset !important;
}

.search-icon {
  opacity: settings.$list-item-icon-opacity;
}

.call-blink {
  animation: blink 0.5s infinite;
}

@keyframes blink {
  50% { opacity: 30% }
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
