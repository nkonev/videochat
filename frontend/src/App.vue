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
                         :model-value="showVideoBadge"
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
          </template>

          <v-btn v-if="chatStore.showGoToBlogButton && !isMobile()" icon :href="goToBlogLink()" :title="$vuetify.locale.t('$vuetify.go_to_blog_post')">
            <v-icon>mdi-postage-stamp</v-icon>
          </v-btn>

          <v-btn v-if="shouldShowFileUpload" icon @click="onShowFileUploadClicked()" :title="$vuetify.locale.t('$vuetify.show_upload_files')">
              {{ chatStore.fileUploadOverallProgress + "%" }}
          </v-btn>

        <template v-if="showSearchButton">
          <v-badge
              :color="getTetATetBadgeColor()"
              dot
              location="right bottom"
              overlap
              bordered
              :model-value="showTetATetBadge"
          >
            <img v-if="shouldShowAvatar() && !isMobile()" @click="onChatAvatarClick()" class="ml-2 v-avatar chat-avatar" :src="chatStore.avatar"/>
          </v-badge>
          <div class="d-flex flex-column app-title mx-2" :class="isInChat() ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked()" :style="{'cursor': isInChat() ? 'pointer' : 'default'}">
            <div :class="!isMobile() ? ['align-self-center'] : []" class="app-title-text" v-html="getTitle()"></div>
            <div v-if="shouldShowSubtitle()" :class="!isMobile() ? ['align-self-center'] : []" class="app-title-subtext">
              {{ getSubtitle() }}
            </div>
          </div>
        </template>

        <template v-if="shouldShowSearch">
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

          <v-snackbar v-model="chatStore.showAlert" :color="chatStore.errorColor" :timeout="chatStore.alertTimeout ? chatStore.alertTimeout : -1" :transition="false">
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

          <!-- "if" is to fix rare issue with snackbar in case background tab in Firefox - it doesn't react on 'removing' or '' status -->
          <v-snackbar v-if="invitedVideoChatAlert" v-model="invitedVideoChatState" color="success" timeout="-1" :multi-line="true" :transition="false">
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
        <ChooseSmileyModal/>
        <ChooseColorModal/>
        <PublishedMessagesModal/>
        <SetPasswordModal/>
      </v-main>

    <v-navigation-drawer :location="isMobile() ? 'left' : 'right'" v-model="chatStore.showDrawer">
        <RightPanelActions/>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import {
  getBlogLink, getExtendedUserFragment,
  getNotificationSubtitle, getNotificationTitle,
  hasLength,
  isCalling,
  isChatRoute,
  setLanguageToVuetify, stopCall, unescapeHtml, goToPreservingQuery
} from "@/utils";
import {
    chat_list_name,
    confirmation_pending_name,
    forgot_password_name, check_email_name, password_restore_enter_new_name,
    registration_name,
    videochat_name, registration_resend_email_name
} from "@/router/routes";
import axios from "axios";
import bus, {
  CHAT_ADD,
  CHAT_DELETED,
  CHAT_EDITED,
  CHAT_REDRAW,
  LOGGED_OUT,
  NOTIFICATION_ADD,
  NOTIFICATION_DELETE,
  OPEN_FILE_UPLOAD_MODAL,
  OPEN_PARTICIPANTS_DIALOG,
  OPEN_PERMISSIONS_WARNING_MODAL,
  PLAYER_MODAL,
  PROFILE_SET,
  REFRESH_ON_WEBSOCKET_RESTORED,
  UNREAD_MESSAGES_CHANGED,
  CO_CHATTED_PARTICIPANT_CHANGED,
  VIDEO_CALL_INVITED,
  VIDEO_CALL_SCREEN_SHARE_CHANGED,
  VIDEO_CALL_USER_COUNT_CHANGED,
  VIDEO_DIAL_STATUS_CHANGED,
  VIDEO_RECORDING_CHANGED,
  WEBSOCKET_RESTORED,
  ON_WINDOW_RESIZED,
  NOTIFICATION_CLEAR_ALL,
  WEBSOCKET_LOST,
  WEBSOCKET_CONNECTED,
  NOTIFICATION_COUNT_CHANGED,
} from "@/bus/bus";
import LoginModal from "@/LoginModal.vue";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'
import {
  searchStringFacade,
  SEARCH_MODE_CHATS,
  SEARCH_MODE_MESSAGES,
  SEARCH_MODE_USERS,
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
import ChooseSmileyModal from "@/ChooseSmileyModal.vue";
import {
    getStoredLanguage, NOTIFICATION_TYPE_ANSWERS,
    NOTIFICATION_TYPE_CALL,
    NOTIFICATION_TYPE_MENTIONS, NOTIFICATION_TYPE_MISSED_CALLS,
    NOTIFICATION_TYPE_NEW_MESSAGES, NOTIFICATION_TYPE_REACTIONS
} from "@/store/localStore";
import ChooseColorModal from "@/ChooseColorModal.vue";
import PublishedMessagesModal from "@/PublishedMessagesModal.vue";
import {createBrowserNotificationIfPermitted, removeBrowserNotification} from "@/browserNotifications.js";
import {getHumanReadableDate} from "@/date.js";
import onFocusMixin from "@/mixins/onFocusMixin.js";
import SetPasswordModal from "@/SetPasswordModal.vue";

const audio = new Audio(`${prefix}/call.mp3`);

const getGlobalEventsData = (message) => {
  return message.data?.globalEvents
};

export default {
    mixins: [
        searchStringFacade(),
        onFocusMixin(),
    ],
    data() {
        return {
            showSearchButton: true,
            invitedVideoChatId: 0,
            invitedVideoChatName: null,
            invitedVideoChatAlert: false,
            invitedVideoChatState: false,

            globalEventsSubscription: null,
            selfProfileEventsSubscription: null,
            showNotificationBadge: false,
            showVideoBadge: false,
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
          if (this.isInChat()) {
              return this.$route.params.id
          } else {
              return null
          }
        },
        notificationsCount() {
            return this.chatStore.notificationsCount
        },
        shouldShowFileUpload() {
            return !!this.chatStore.fileUploadingQueue.length
        },
        showTetATetBadge() {
            return this.chatStore.oppositeUserOnline && !!(this.chatStore.chatDto?.tetATet) && hasLength(this.chatStore.chatDto?.avatar) && !this.isMobile()
        },
        shouldShowSearch() {
            return this.chatStore.isShowSearch && !(this.isVideoRoute() && !this.chatStore.videoMessagesEnabled)
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
        getTetATetBadgeColor() {
          if (this.chatStore.oppositeUserInVideo) {
            return 'red'
          } else {
            return 'green'
          }
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
            goToPreservingQuery(this.$route, this.$router, routerNewState);
        },
        stopCall() {
          stopCall(this.chatStore, this.$route, this.$router);
        },

        onProfileSet(){
            this.chatStore.fetchNotificationsCount();
            this.chatStore.fetchHasNewMessages();
            this.refreshInvitationCall();
            this.globalEventsSubscription.graphQlSubscribe();
            this.selfProfileEventsSubscription.graphQlSubscribe();
        },
        onLoggedOut() {
            this.resetVariables();
            this.globalEventsSubscription.graphQlUnsubscribe();
            this.selfProfileEventsSubscription.graphQlUnsubscribe();
        },
        resetVariables() {
            this.resetVideoInvitation()
            this.chatStore.unsetNotificationsAndHasNewMessages();
        },
        canSwitchSearchType() {
            return this.isInChat() || this.$route.name == chat_list_name
        },
        switchSearchType() {
          this.chatStore.switchSearchType()
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
        getGlobalGraphQlSubscriptionQuery() {
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
                          shortInfo
                          loginColor
                        }
                        canResend
                        availableToSearch
                        isResultFromSearch
                        pinned
                        blog
                        loginColor
                        regularParticipantCanPublishMessage
                        lastSeenDateTime
                        regularParticipantCanPinMessage
                        blogAbout
                        regularParticipantCanWriteMessage
                        canWriteMessage
                      }
                      chatDeletedEvent {
                        id
                      }
                      coChattedParticipantEvent {
                        id
                        login
                        avatar
                        shortInfo
                        loginColor
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
                        avatar
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
                        notificationDto {
                          id
                          chatId
                          messageId
                          notificationType
                          description
                          createDateTime
                          byUserId
                          byLogin
                          byAvatar
                          chatTitle
                        }
                        count
                      }
                      forceLogout {
                        reasonType
                      }
                      hasUnreadMessagesChanged {
                        hasUnreadMessages
                      }
                      browserNotification {
                        chatId
                        chatName
                        chatAvatar
                        messageId
                        messageText
                        ownerId
                        ownerLogin
                      }
                    }
                  }
              `
        },
        onNextGlobalSubscriptionElement(e) {
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
            const d = getGlobalEventsData(e).coChattedParticipantEvent;
            bus.emit(CO_CHATTED_PARTICIPANT_CHANGED, d);
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
            this.processNotificationAsInBrowser(d.notificationDto, true);
          } else if (getGlobalEventsData(e).eventType === 'notification_delete') {
              const d = getGlobalEventsData(e).notificationEvent;
              bus.emit(NOTIFICATION_DELETE, d);
              this.processNotificationAsInBrowser(d.notificationDto, false);
          } else if (getGlobalEventsData(e).eventType === 'notification_clear_all') {
              const d = getGlobalEventsData(e).notificationEvent;
              bus.emit(NOTIFICATION_CLEAR_ALL, d);
              this.processClearAllNotificationsInBrowser(d);
          } else if (getGlobalEventsData(e).eventType === 'has_unread_messages_changed') {
              const d = getGlobalEventsData(e).hasUnreadMessagesChanged;
              this.chatStore.setHasNewMessages(d.hasUnreadMessages);
          } else if (getGlobalEventsData(e).eventType === 'browser_notification_add_message') {
              const d = getGlobalEventsData(e).browserNotification;
              createBrowserNotificationIfPermitted(this.$router, d.chatId, d.chatName, d.chatAvatar, d.messageId, d.messageText, NOTIFICATION_TYPE_NEW_MESSAGES);
          } else if (getGlobalEventsData(e).eventType === 'browser_notification_remove_message') {
              removeBrowserNotification(NOTIFICATION_TYPE_NEW_MESSAGES);
          } else if (getGlobalEventsData(e).eventType === 'user_sessions_killed') {
            const d = getGlobalEventsData(e).forceLogout;
            console.log("Killed sessions, reason:", d.reasonType)
            this.chatStore.unsetUser();
            bus.emit(LOGGED_OUT);
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
            bus.emit(OPEN_PARTICIPANTS_DIALOG, {chatId: this.chatId});
          }
        },
        onShowFileUploadClicked() {
            bus.emit(OPEN_FILE_UPLOAD_MODAL, { });
        },
        onFocus() {
          if (this.chatStore.currentUser) {
            this.chatStore.fetchNotificationsCount();
            this.chatStore.fetchHasNewMessages();
            this.refreshInvitationCall();
          }
        },
        refreshInvitationCall() {
          axios.get(`/api/video/user/being-invited-status`, {
              params: {
                  tokenId: this.chatStore.videoTokenId
              }
          }).then(({data}) => {
            this.onVideoCallInvited(data);
          })
        },
        shouldShowAvatar() {
            return hasLength(this.chatStore.avatar)
        },
        getTitle() {
            let bldr = this.chatStore.title;
            if (!this.shouldShowAvatar()) {
              if (this.chatStore.oppositeUserOnline) {
                bldr += " (" + this.$vuetify.locale.t('$vuetify.user_online') + ")";
              }
              if (this.chatStore.oppositeUserInVideo) {
                bldr += " (" + this.$vuetify.locale.t('$vuetify.user_in_video_call') + ")";
              }
            }
            return bldr
        },
        getSubtitle() {
            if (!!this.chatStore.moreImportantSubtitleInfo) {
              return this.chatStore.moreImportantSubtitleInfo
            } if (!!this.chatStore.usersWritingSubtitleInfo) {
              return this.chatStore.usersWritingSubtitleInfo
            } else if (this.chatStore.oppositeUserLastSeenDateTime) {
                return this.$vuetify.locale.t('$vuetify.last_seen_at', getHumanReadableDate(this.chatStore.oppositeUserLastSeenDateTime));
            } else {
                return this.chatStore.chatUsersCount + " " + this.$vuetify.locale.t('$vuetify.participants')
            }
        },
        shouldShowSubtitle() {
            return !!this.chatStore.chatUsersCount || !!this.chatStore.moreImportantSubtitleInfo || !!this.chatStore.usersWritingSubtitleInfo || this.chatStore.oppositeUserLastSeenDateTime
        },
        afterRouteInitialized() {
            return this.chatStore.fetchAvailableOauth2Providers()
        },
        fetchProfileIfNeed() {
            if (!this.chatStore.currentUser) {
                if (this.$route.name == registration_name || this.$route.name == confirmation_pending_name || this.$route.name == forgot_password_name || this.$route.name == password_restore_enter_new_name || this.$route.name == check_email_name || this.$route.name == confirmation_pending_name || this.$route.name == registration_resend_email_name) {
                    return
                }
                this.chatStore.fetchUserProfile().then(()=>{
                    setLanguageToVuetify(this, getStoredLanguage());
                })
            }
        },
        onWsLost() {
          this.chatStore.moreImportantSubtitleInfo = this.$vuetify.locale.t('$vuetify.connecting');
        },
        onWsConnected() {
          this.chatStore.moreImportantSubtitleInfo = null;
        },
        onWsRestored() {
          console.warn("REFRESH_ON_WEBSOCKET_RESTORED auto");
          bus.emit(REFRESH_ON_WEBSOCKET_RESTORED);
        },
        resetVideoInvitation() {
            this.invitedVideoChatState = false;
            this.$nextTick(()=>{
              this.invitedVideoChatAlert = false;
              this.invitedVideoChatId = 0;
              this.invitedVideoChatName = null;
              removeBrowserNotification(NOTIFICATION_TYPE_CALL);
            })
        },
        onVideoCallInvited(data) {
          if (isCalling(data.status)) {
              this.invitedVideoChatAlert = true;
              this.$nextTick(()=>{
                this.invitedVideoChatId = data.chatId;
                this.invitedVideoChatName = unescapeHtml(data.chatName);
                this.invitedVideoChatState = true;
              }).then(()=>{
                  createBrowserNotificationIfPermitted(this.$router, data.chatId, data.chatName, data.avatar, null, this.$vuetify.locale.t('$vuetify.you_called_short', this.invitedVideoChatId), NOTIFICATION_TYPE_CALL);
                  audio.play().catch(error => {
                      console.warn("Unable to play sound", error);
                      bus.emit(OPEN_PERMISSIONS_WARNING_MODAL);
                  })
              })

          } else {
            this.resetVideoInvitation();
          }
        },
        onClickInvitation() {
          const routerNewState = { name: videochat_name, params: { id: this.invitedVideoChatId }};
          this.invitedVideoChatState = false;
          this.$router.push(routerNewState).then(()=>{
            this.resetVideoInvitation();
          })
        },
        onClickCancelInvitation() {
          axios.put(`/api/video/${this.invitedVideoChatId}/dial/cancel`).then(()=>{
            this.resetVideoInvitation();
          });
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
        onWindowResized() {
            bus.emit(ON_WINDOW_RESIZED)
        },
        processNotificationAsInBrowser(item, add) {

            const title = getNotificationTitle(item);
            const subtitle = getNotificationSubtitle(this.$vuetify, item);

            let type;
            switch(item.notificationType) {
                case "missed_call":
                    type = NOTIFICATION_TYPE_MISSED_CALLS;
                    break
                case "mention":
                    type = NOTIFICATION_TYPE_MENTIONS;
                    break
                case "reply":
                    type = NOTIFICATION_TYPE_ANSWERS;
                    break
                case "reaction":
                    type = NOTIFICATION_TYPE_REACTIONS;
                    break
            }

            if (add) {
                createBrowserNotificationIfPermitted(this.$router, item.chatId, title, item.byAvatar, item.messageId, subtitle, type);
            } else {
                removeBrowserNotification(type);
            }
        },
        processClearAllNotificationsInBrowser(dto) {
            removeBrowserNotification(NOTIFICATION_TYPE_MENTIONS);
            removeBrowserNotification(NOTIFICATION_TYPE_MISSED_CALLS);
            removeBrowserNotification(NOTIFICATION_TYPE_ANSWERS);
            removeBrowserNotification(NOTIFICATION_TYPE_REACTIONS);
            removeBrowserNotification(NOTIFICATION_TYPE_NEW_MESSAGES);
            removeBrowserNotification(NOTIFICATION_TYPE_CALL);
        },

        getUserIdsSubscribeTo() {
          const ret = [];
          if (this.chatStore.currentUser) {
            ret.push(this.chatStore.currentUser.id)
          }
          return ret;
        },
        getSelfGraphQlSubscriptionQuery() {
          return `
                    subscription {
                      userAccountEvents(userIdsFilter: ${this.getUserIdsSubscribeTo()}) {
                        userAccountEvent {
                          ${getExtendedUserFragment(true)},
                          ... on UserDeletedDto {
                            id
                          }
                        }
                        eventType
                      }
                    }
                `
        },
        onSelfNextSubscriptionElement(e) {
          const d = e.data?.userAccountEvents;
          if (d.eventType === 'user_account_changed') {
            this.onEditUser(d.userAccountEvent);
          }
        },
        onEditUser(u) {
          this.chatStore.currentUser = u;
        },
        updateNotificationBadge() {
          this.showNotificationBadge = this.chatStore.notificationsCount != 0 && !this.chatStore.showDrawer
        },
        updateVideoBadge() {
          this.showVideoBadge = !!parseInt(this.chatStore.videoChatUsersCount)
        },
        // needed to update video badge after /api/video/${chatId}/users was called by FOCUS event
        onVideoCallChanged(dto) {
          if (dto.chatId == this.chatId) {
            this.chatStore.videoChatUsersCount = dto.usersCount;
            this.$nextTick(()=>{
              console.debug("For", dto, "updating updateVideoBadge with", this.chatStore.videoChatUsersCount);
              this.updateVideoBadge();
            })
          }
        },
        onNotificationCountChanged() {
          this.updateNotificationBadge();
        },
    },
    components: {
        ChooseColorModal,
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
        ChooseSmileyModal,
        PublishedMessagesModal,
        SetPasswordModal,
    },
    watch: {
      'chatStore.notificationsCount': {
        handler: function (newValue, oldValue) {
            this.updateNotificationBadge()
        }
      },
      'chatStore.showDrawer': {
        handler: function (newValue, oldValue) {
          this.updateNotificationBadge()
        }
      },
      'chatStore.videoChatUsersCount': {
        handler: function (newValue, oldValue) {
          this.updateVideoBadge()
        }
      },
    },

    created() {
        this.afterRouteInitialized = once(this.afterRouteInitialized);
    },
    mounted() {
        window.addEventListener("resize", this.onWindowResized);

        createGraphQlClient();

        // create subscription object before ON_PROFILE_SET (afterRouteInitialized())
        this.globalEventsSubscription = graphqlSubscriptionMixin('globalEvents', this.getGlobalGraphQlSubscriptionQuery, this.setErrorSilent, this.onNextGlobalSubscriptionElement);
        this.selfProfileEventsSubscription = graphqlSubscriptionMixin('userSelfProfileEvents', this.getSelfGraphQlSubscriptionQuery, this.setErrorSilent, this.onSelfNextSubscriptionElement);

        // place onProfileSet() before fetchProfileIfNeed() to start subscription in onProfileSet()
        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);
        bus.on(WEBSOCKET_CONNECTED, this.onWsConnected);
        bus.on(WEBSOCKET_LOST, this.onWsLost);
        bus.on(WEBSOCKET_RESTORED, this.onWsRestored);
        bus.on(VIDEO_CALL_INVITED, this.onVideoCallInvited);
        bus.on(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
        bus.on(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
        bus.on(NOTIFICATION_COUNT_CHANGED, this.onNotificationCountChanged);

        // To trigger fetching profile that 's going to trigger starting subscriptions
        // It's placed after each route in order not to have a race-condition
        this.$router.afterEach((to, from) => {
          this.afterRouteInitialized().then(()=>{
            this.fetchProfileIfNeed();
          })
        });

        this.installOnFocus();
    },
    beforeUnmount() {
        this.uninstallOnFocus();
        window.removeEventListener("resize", this.onWindowResized);

        bus.off(PROFILE_SET, this.onProfileSet);
        bus.off(LOGGED_OUT, this.onLoggedOut);
        bus.off(WEBSOCKET_CONNECTED, this.onWsConnected);
        bus.off(WEBSOCKET_LOST, this.onWsLost);
        bus.off(WEBSOCKET_RESTORED, this.onWsRestored);
        bus.off(VIDEO_CALL_INVITED, this.onVideoCallInvited);
        bus.off(VIDEO_RECORDING_CHANGED, this.onVideRecordingChanged);
        bus.off(VIDEO_CALL_USER_COUNT_CHANGED, this.onVideoCallChanged);
        bus.off(NOTIFICATION_COUNT_CHANGED, this.onNotificationCountChanged);

        this.globalEventsSubscription.graphQlUnsubscribe();
        this.selfProfileEventsSubscription.graphQlUnsubscribe();
        this.globalEventsSubscription = null;
        this.selfProfileEventsSubscription = null;

        destroyGraphqlClient();
    },
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

.v-card {
  .v-pagination__list {
    justify-content: start;
  }
}

// reverts some changes from ~3.7.0 (from F12)
.my-actions .v-btn ~ .v-btn:not(.v-btn-toggle .v-btn) {
  margin-inline-start: .5rem;
}

// see also dialogModal.styl

</style>

<style lang="stylus">
@import "constants.styl"

.colored-link {
    color: $linkColor;
    text-decoration none
}

.gray-link {
    color: $grayColor;
    text-decoration none
}

.nodecorated-link {
    text-decoration none
}

.with-space {
  white-space: pre;
}

.with-ellipsis {
    overflow:hidden;
    text-overflow: ellipsis;
}

.list-item-prepend-spacer {
    .v-list-item__prepend {
        .v-list-item__spacer {
            width: 12px
        }
    }
}

div .stop-scrolling {
    overflow: hidden !important;
}

.inline-caption-base {
  z-index 2
  display inherit
  margin: 0;
  left 0.4em
  bottom 0.4em
  position: absolute
  background rgba(255, 255, 255, 0.65)
  padding-left 0.3em
  padding-right 0.3em
  border-radius 4px
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size 0.9rem
}

.caption-small {
  color:rgba(0, 0, 0, .6);
  font-size: 0.9rem;
  font-weight: 500;
  line-height: 1rem;
  display: inherit
}

</style>

<style lang="stylus" scoped>
.chat-avatar {
  display: block;
  max-width: 36px;
  max-height: 36px;
  width: auto;
  height: auto;
  cursor: pointer
}

</style>
