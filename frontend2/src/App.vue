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
                      <v-icon :x-large="isMobile()" :class="chatStore.shouldPhoneBlink ? 'call-blink' : 'red--text'">mdi-phone</v-icon>
                  </v-btn>
              </v-badge>


          </template>

          <v-btn v-if="chatStore.showScrollDown" icon @click="scrollDown()" :title="$vuetify.locale.t('$vuetify.scroll_down')">
            <v-icon :x-large="isMobile()">mdi-arrow-down-thick</v-icon>
          </v-btn>

          <v-spacer></v-spacer>
          <img v-if="!!chatStore.avatar && !isMobile()" @click="onChatAvatarClick()" class="v-avatar chat-avatar" :src="chatStore.avatar"/>
          <div color="white" class="d-flex flex-column px-2 app-title" :class="chatId ? 'app-title-hoverable' : 'app-title'" @click="onInfoClicked()" :style="{'cursor': chatId ? 'pointer' : 'default'}">
            <div :class="!isMobile() ? ['align-self-center'] : []" class="app-title-text" v-html="chatStore.title"></div>
            <div v-if="!!chatStore.chatUsersCount" :class="!isMobile() ? ['align-self-center'] : []" class="app-title-subtext">
              {{ chatStore.chatUsersCount }} {{ $vuetify.locale.t('$vuetify.participants') }}</div>
          </div>
          <v-spacer></v-spacer>

          <v-card variant="plain" min-width="330" v-if="chatStore.isShowSearch" style="margin-left: 1.2em; margin-right: 2px">
            <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line v-model="searchStringFacade" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" :label="searchName()">
              <template v-slot:append-inner>
                <v-btn icon density="compact" @click.prevent="switchSearchType()"><v-icon class="search-icon">{{ searchIcon }}</v-icon></v-btn>
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
    </v-main>

    <v-navigation-drawer location="right" v-model="chatStore.drawer">
        <RightPanelActions/>
    </v-navigation-drawer>
  </v-app>
</template>

<script>
import 'typeface-roboto'; // More modern versions turn out into almost non-bold font in Firefox
import { hasLength} from "@/utils";
import { chat_name, videochat_name} from "@/router/routes";
import axios from "axios";
import bus, {
  CHAT_ADD,
  CHAT_DELETED,
  CHAT_EDITED,
  LOGGED_OUT, OPEN_PARTICIPANTS_DIALOG, PLAYER_MODAL,
  PROFILE_SET,
  SCROLL_DOWN, UNREAD_MESSAGES_CHANGED, VIDEO_CALL_INVITED, VIDEO_CALL_SCREEN_SHARE_CHANGED,
  VIDEO_CALL_USER_COUNT_CHANGED, VIDEO_DIAL_STATUS_CHANGED, VIDEO_RECORDING_CHANGED,
} from "@/bus/bus";
import LoginModal from "@/LoginModal.vue";
import {useChatStore} from "@/store/chatStore";
import { mapStores } from 'pinia'
import {searchStringFacade, SEARCH_MODE_CHATS, SEARCH_MODE_MESSAGES} from "@/mixins/searchString";
import RightPanelActions from "@/RightPanelActions.vue";
import SettingsModal from "@/SettingsModal.vue";
import SimpleModal from "@/SimpleModal.vue";
import {createGraphQlClient, destroyGraphqlClient} from "@/graphql/graphql";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";

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
            return this.$route.params.id
        },
        notificationsCount() {
            return this.chatStore.notifications.length
        },
        showNotificationBadge() {
            return this.notificationsCount != 0 && !this.chatStore.drawer
        },
        searchIcon() {
          if (this.chatStore.searchType == SEARCH_MODE_CHATS) {
            return 'mdi-forum'
          } else if (this.chatStore.searchType == SEARCH_MODE_MESSAGES) {
            return 'mdi-message-text-outline'
          }
        },
    },
    methods: {
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
            this.chatStore.drawer = !this.chatStore.drawer;
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

        onProfileSet(){
            this.chatStore.fetchNotifications();
            this.graphQlSubscribe();
        },
        onLoggedOut() {
            this.resetVariables();
            this.graphQlUnsubscribe();
        },
        resetVariables() {
            this.chatStore.unsetNotifications();
        },
        switchSearchType() {
          this.chatStore.switchSearchType()
        },
        scrollDown () {
          bus.emit(SCROLL_DOWN)
        },
        searchName() {
            if (this.chatStore.searchType == SEARCH_MODE_CHATS) {
              return this.$vuetify.locale.t('$vuetify.search_in_chats')
            } else if (this.chatStore.searchType == SEARCH_MODE_MESSAGES) {
              return this.$vuetify.locale.t('$vuetify.search_in_messages')
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
            bus.emit(UNREAD_MESSAGES_CHANGED, d);//
          } else if (getGlobalEventsData(e).eventType === 'notification_add') {
            const d = getGlobalEventsData(e).notificationEvent;
            this.chatStore.notificationAdd(d);
          } else if (getGlobalEventsData(e).eventType === 'notification_delete') {
            const d = getGlobalEventsData(e).notificationEvent;
            this.chatStore.notificationDelete(d);
          }
        },
        onChatAvatarClick() {
          bus.emit(PLAYER_MODAL, {"canShowAsImage": true, url: this.chatAvatar})
        },
        onInfoClicked() {
          if (this.chatId) {
            bus.emit(OPEN_PARTICIPANTS_DIALOG, this.chatId);
          }
        },

    },
    components: {
        RightPanelActions,
        LoginModal,
        SettingsModal,
        SimpleModal,
    },
    created() {
        createGraphQlClient();

        bus.on(PROFILE_SET, this.onProfileSet);
        bus.on(LOGGED_OUT, this.onLoggedOut);

        this.chatStore.fetchAvailableOauth2Providers().then(() => {
            this.chatStore.fetchUserProfile();
        })
    },

    beforeUnmount() {
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

  &-text {
    font-size: .875rem;
    font-weight: 500;
    letter-spacing: .0892857143em;
    text-indent: .0892857143em;
  }

  &-subtext {
    font-size: .7rem;
    letter-spacing: initial;
    text-transform: initial;
    opacity: 50%
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

</style>
