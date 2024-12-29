<template>
  <v-container v-if="viewableUser" fluid
  >
    <v-container class="d-flex justify-space-around flex-column py-0 user-self-settings-container" fluid>
      <v-card-title class="title px-0 pb-0">
        {{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ viewableUser.id }}
      </v-card-title>
      <v-img v-if="hasAva"
         :src="ava"
         max-width="320"
         class="mt-2"
      >
      </v-img>
      <span class="d-flex flex-wrap">
        <span class="text-h3" :style="getLoginColoredStyle(viewableUser)" v-html="getUserNamePretty()">
        </span>

        <span class="ml-2 mb-2 d-flex flex-row align-self-end">
          <span v-if="online" class="text-grey d-flex flex-row">
            <v-icon :color="getUserBadgeColor(this)">mdi-checkbox-marked-circle</v-icon>
            <span class="ml-1">
              {{ isInVideo ? $vuetify.locale.t('$vuetify.user_in_video_call') : $vuetify.locale.t('$vuetify.user_online') }}
            </span>
          </span>
          <span v-else class="text-grey d-flex flex-row">
            <v-icon>mdi-checkbox-marked-circle</v-icon>
            <span class="ml-1">
              {{ $vuetify.locale.t('$vuetify.user_offline') }}
            </span>
          </span>
        </span>
      </span>

      <v-card-subtitle class="title px-0 pb-0" v-if="viewableUser.lastSeenDateTime">
        {{ $vuetify.locale.t('$vuetify.last_seen_at', getHumanReadableDate(viewableUser.lastSeenDateTime)) }}
      </v-card-subtitle>

      <template v-if="displayShortInfo(viewableUser)">
          <span class="mx-0 my-1 force-wrap" v-html="viewableUser.shortInfo"></span>
      </template>

      <v-container class="ma-0 pa-0 pt-1">
        <v-btn color="primary" @click="tetATet(viewableUser.id)">
          <template v-slot:prepend><v-icon>mdi-message-text-outline</v-icon></template>
          <template v-slot:default>
            {{ $vuetify.locale.t('$vuetify.user_open_chat') }}
          </template>
        </v-btn>

        <v-btn class="ml-2" variant="plain" @click="onShowContextMenu" icon="mdi-menu"/>
      </v-container>
    </v-container>

    <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.roles') }}</v-card-title>
    <v-card-actions class="mx-2 nominheight">
        <v-chip v-for="(role, index) in viewableUser?.additionalData?.roles"
          density="comfortable"
          text-color="white"
        >
          <template v-slot:default>
              <span>
                {{role}}
              </span>
          </template>
        </v-chip>
    </v-card-actions>

    <v-card-title class="title pb-0 pt-1" v-if="viewableUser.ldap">LDAP</v-card-title>
    <v-chip
          density="comfortable"
          v-if="viewableUser.ldap"
          class="mx-4 c-btn-database"
          text-color="white"
    >
          <template v-slot:prepend>
              <font-awesome-icon :icon="{ prefix: 'fas', iconName: 'database'}"></font-awesome-icon>
          </template>
          <template v-slot:default>
              <span>
                Ldap
              </span>
          </template>
    </v-chip>

    <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.bound_oauth2_providers') }}</v-card-title>
    <v-card-actions class="mx-2" v-if="shouldShowBound()">
      <v-chip
        v-if="viewableUser.oauth2Identifiers.vkontakteId"
        min-width="80px"
        label
        class="c-btn-vk py-5"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.facebookId"
        min-width="80px"
        label
        class="c-btn-fb py-5"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.googleId"
        min-width="80px"
        label
        class="c-btn-google py-5"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.keycloakId"
        min-width="80px"
        label
        class="c-btn-keycloak py-5"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'key'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

    </v-card-actions>

    <UserListContextMenu
          ref="contextMenuRef"
          @tetATet="this.tetATetUser"
          @unlockUser="this.unlockUser"
          @lockUser="this.lockUser"
          @unconfirmUser="this.unconfirmUser"
          @confirmUser="this.confirmUser"
          @deleteUser="this.deleteUser"
          @changeRole="this.changeRole"
          @removeSessions="this.removeSessions"
          @enableUser="this.enableUser"
          @disableUser="this.disableUser"
          @setPassword="this.setPassword"
    >
    </UserListContextMenu>
    <UserRoleModal/>
  </v-container>
</template>

<script>

import axios from "axios";
import {chat_name, profile_list_name} from "./router/routes";
import {
  deepCopy,
  getExtendedUserFragment,
  getLoginColoredStyle,
  hasLength,
  isStrippedUserLogin,
  setTitle
} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import userStatusMixin from "@/mixins/userStatusMixin";
import bus, {
  CHANGE_ROLE_DIALOG,
  CLOSE_SIMPLE_MODAL,
  LOGGED_OUT, OPEN_SET_PASSWORD_MODAL,
  OPEN_SIMPLE_MODAL,
  PROFILE_SET
} from "@/bus/bus";
import {getHumanReadableDate} from "@/date.js";
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin.js";
import UserListContextMenu from "@/UserListContextMenu.vue";
import UserRoleModal from "@/UserRoleModal.vue";
import onFocusMixin from "@/mixins/onFocusMixin.js";

export default {
  components: {
      UserRoleModal,
      UserListContextMenu,
  },
  mixins: [
      userStatusMixin('userStatusInUserProfile'), // another subscription
      onFocusMixin(),
  ],
  data() {
    return {
      viewableUser: null,
      online: false,
      isInVideo: false,
      userProfileEventsSubscription: null,
    }
  },
  computed: {
    ...mapStores(useChatStore),
    userId() {
      return this.$route.params.id
    },
    ava() {
      const maybeUser = this.viewableUser;
      if (maybeUser) {
        if (hasLength(maybeUser.avatarBig)) {
          return maybeUser.avatarBig
        } else if (hasLength(maybeUser.avatar)) {
          return maybeUser.avatar
        } else {
          return null
        }
      }
    },
    hasAva() {
      const maybeUser = this.viewableUser;
      return hasLength(maybeUser?.avatarBig) || hasLength(maybeUser?.avatar)
    },
  },
  methods: {
    getLoginColoredStyle,
    getHumanReadableDate,
    getUserNamePretty() {
      if (isStrippedUserLogin(this.viewableUser)) {
          return "<s>" + this.viewableUser?.login + "</s>"
      } else {
          return this.viewableUser?.login
      }
    },
    loadUser() {
      this.viewableUser = null;
      axios.get(`/api/aaa/user/${this.userId}`, {
        signal: this.requestAbortController.signal
      }).then((response) => {
            if (response.status == 204) {
              this.$router.push(({name: profile_list_name}));
              this.setWarning(this.$vuetify.locale.t('$vuetify.user_not_found'));
            } else {
              this.viewableUser = response.data;
            }
      })
    },
    unlockUser(user) {
      axios.post('/api/aaa/user/lock', {userId: user.id, lock: false}, {
        signal: this.requestAbortController.signal
      });
    },
    lockUser(user) {
      axios.post('/api/aaa/user/lock', {userId: user.id, lock: true}, {
        signal: this.requestAbortController.signal
      });
    },
    unconfirmUser(user) {
      axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: false}, {
        signal: this.requestAbortController.signal
      });
    },
    confirmUser(user) {
      axios.post('/api/aaa/user/confirm', {userId: user.id, confirm: true}, {
        signal: this.requestAbortController.signal
      });
    },
    deleteUser(user) {
      bus.emit(OPEN_SIMPLE_MODAL, {
          buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
          title: this.$vuetify.locale.t('$vuetify.delete_user_title', user.id),
          text: this.$vuetify.locale.t('$vuetify.delete_user_text', user.login),
          actionFunction: (that) => {
              that.loading = true;
              axios.delete('/api/aaa/user', {
                  params: {
                      userId: user.id
                  },
                  signal: this.requestAbortController.signal
              }).then(() => {
                  bus.emit(CLOSE_SIMPLE_MODAL);
              }).finally(()=>{
                  that.loading = false;
              })
          }
      });
    },
    changeRole(user) {
      bus.emit(CHANGE_ROLE_DIALOG, user)
    },
    removeSessions(user) {
      axios.delete('/api/aaa/sessions', {
          params: {
              userId: user.id
          },
          signal: this.requestAbortController.signal
      });
    },
    tetATetUser(user) {
      this.tetATet(user.id)
    },
    tetATet(withUserId) {
      axios.put(`/api/chat/tet-a-tet/${withUserId}`, {
        signal: this.requestAbortController.signal
      }).then(response => {
        this.$router.push(({ name: chat_name, params: { id: response.data.id}}));
      })
    },

    onUserStatusChanged(dtos) {
      if (dtos) {
        dtos?.forEach(dtoItem => {
          if (dtoItem.online !== null && this.userId == dtoItem.userId) {
            this.online = dtoItem.online;
          }
          if (dtoItem.isInVideo !== null && this.userId == dtoItem.userId) {
            this.isInVideo = dtoItem.isInVideo;
          }
        })
      }
    },
    getUserIdsSubscribeTo() {
      return [this.userId];
    },
    displayShortInfo(user){
      return hasLength(user.shortInfo)
    },
    shouldShowBound() {
      const copied = deepCopy(this.viewableUser.oauth2Identifiers);
      delete copied["@class"];

      let has = false;
      for (const aProp in copied) {
        if (hasLength(copied[aProp])) {
          has = true;
          break
        }
      }
      return has && (
        this.chatStore.availableOAuth2Providers.includes('vkontakte') ||
        this.chatStore.availableOAuth2Providers.includes('facebook') ||
        this.chatStore.availableOAuth2Providers.includes('google') ||
        this.chatStore.availableOAuth2Providers.includes('keycloak')
      )
    },
    setMainTitle() {
      const aTitle = this.$vuetify.locale.t('$vuetify.user_profile');
      this.chatStore.title = aTitle;
      setTitle(aTitle);
    },
    unsetMainTitle() {
      const aTitle = null;
      this.chatStore.title = aTitle;
      setTitle(aTitle);
    },
    onLoggedOut() {
      this.graphQlUserStatusUnsubscribe();
      this.userProfileEventsSubscription.graphQlUnsubscribe();
    },
    onProfileSet() {
      this.loadUser();
      this.graphQlUserStatusSubscribe();
      this.userProfileEventsSubscription.graphQlSubscribe();
      this.requestStatuses();
    },
    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
    getGraphQlSubscriptionQuery() {
          return `
                subscription {
                  userAccountEvents(userIdsFilter: ${this.getUserIdsSubscribeTo()}) {
                    userAccountEvent {
                      ${getExtendedUserFragment()},
                      ... on UserDeletedDto {
                        id
                      }
                    }
                    eventType
                  }
                }
            `
    },
    onNextSubscriptionElement(e) {
          const d = e.data?.userAccountEvents;
          if (d.eventType === 'user_account_changed') {
              this.onEditUser(d.userAccountEvent);
          } else if (d.eventType === 'user_account_deleted') {
              this.onDeleteUser(d.userAccountEvent);
          }
    },
    onDeleteUser(u) {
        this.$router.push({name: profile_list_name})
    },
    onEditUser(u) {
        this.viewableUser = u;
    },
    onFocus() {
          if (this.chatStore.currentUser) {
            this.requestStatuses();
          }
    },
    requestStatuses() {
          this.$nextTick(()=>{
              this.triggerUsesStatusesEvents(this.userId, this.requestAbortController.signal);
          })
    },
    onShowContextMenu(e) {
        if (this.chatStore.currentUser) {
            this.$refs.contextMenuRef.onShowContextMenu(e, this.viewableUser);
        }
    },
    enableUser(user) {
      axios.post('/api/aaa/user/enable', {userId: user.id, enable: true}, {
        signal: this.requestAbortController.signal
      });
    },
    disableUser(user) {
      axios.post('/api/aaa/user/enable', {userId: user.id, enable: false}, {
        signal: this.requestAbortController.signal
      });
    },
    setPassword(user) {
      bus.emit(OPEN_SET_PASSWORD_MODAL, {userId: user.id})
    },
  },
  mounted() {
    bus.on(LOGGED_OUT, this.onLoggedOut);
    bus.on(PROFILE_SET, this.onProfileSet);
    this.setMainTitle();

    // create subscription object before ON_PROFILE_SET
    this.userProfileEventsSubscription = graphqlSubscriptionMixin('userProfileEvents', this.getGraphQlSubscriptionQuery, this.setErrorSilent, this.onNextSubscriptionElement);

    if (this.canDrawUsers()) {
      this.onProfileSet();
    }

    this.installOnFocus();
  },
  beforeUnmount() {
    this.uninstallOnFocus();

    this.graphQlUserStatusUnsubscribe();
    this.userProfileEventsSubscription.graphQlUnsubscribe();
    this.userProfileEventsSubscription = null;

    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(PROFILE_SET, this.onProfileSet);

    this.unsetMainTitle();

    this.viewableUser = null;
    this.online = false;
    this.isInVideo = false;
  },
  watch: {
    '$vuetify.locale.current': {
      handler: function (newValue, oldValue) {
        this.setMainTitle()
      },
    }
  },
}
</script>

<style lang="stylus">
@import "oAuth2.styl"
</style>
<style lang="stylus" scoped>
.nominheight {
    min-height unset
}

.force-wrap {
    word-wrap: break-word;
}
</style>
