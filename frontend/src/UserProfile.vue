<template>
  <v-card v-if="viewableUser"
          class="mr-auto"
          width="fit-content"
  >
    <v-container class="d-flex justify-space-around flex-column py-0 user-self-settings-container">
      <v-card-title class="title px-0 pb-0">
        {{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ viewableUser.id }}      </v-card-title>
      <v-img v-if="hasAva"
         :src="ava"
             max-width="320"
         class="mt-2"
      >
      </v-img>
      <span class="d-flex">
        <span class="text-h3" :style="getLoginColoredStyle(viewableUser)">
          {{ viewableUser.login }}
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
      <v-divider></v-divider>
      <span v-if="viewableUser.email" class="text-h6">{{ viewableUser.email }}</span>
      <v-divider></v-divider>
      <span v-if="displayShortInfo(viewableUser)" class="my-1">{{ viewableUser.shortInfo }}</span>

      <v-container class="ma-0 pa-0">
        <v-btn v-if="isNotMyself()" color="primary" @click="tetATet(viewableUser.id)">
          <template v-slot:prepend><v-icon>mdi-message-text-outline</v-icon></template>
          <template v-slot:default>
            {{ $vuetify.locale.t('$vuetify.user_open_chat') }}
          </template>
        </v-btn>
      </v-container>
    </v-container>

    <v-divider class="mx-4"></v-divider>
    <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.bound_oauth2_providers') }}</v-card-title>
    <v-card-actions class="mx-2" v-if="shouldShowBound()">
      <v-chip
        v-if="viewableUser.oauth2Identifiers.vkontakteId"
        min-width="80px"
        label
        class="c-btn-vk py-5 mr-2"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.facebookId"
        min-width="80px"
        label
        class="c-btn-fb py-5 mr-2"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.googleId"
        min-width="80px"
        label
        class="c-btn-google py-5 mr-2"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

      <v-chip
        v-if="viewableUser.oauth2Identifiers.keycloakId"
        min-width="80px"
        label
        class="c-btn-keycloak py-5 mr-2"
        text-color="white"
      >
        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'key'}" :size="'2x'"></font-awesome-icon>
      </v-chip>

    </v-card-actions>
  </v-card>
</template>

<script>

import axios from "axios";
import {chat_name} from "./router/routes";
import {deepCopy, getLoginColoredStyle, hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import userStatusMixin from "@/mixins/userStatusMixin";
import bus, {LOGGED_OUT, PROFILE_SET} from "@/bus/bus";

export default {
  mixins: [
    userStatusMixin('userProfile')
  ],
  data() {
    return {
      viewableUser: null,
      online: false,
      isInVideo: false,
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
    isNotMyself() {
      return this.chatStore.currentUser && this.chatStore.currentUser.id != this.viewableUser.id
    },
    loadUser() {
      this.viewableUser = null;
      axios.get(`/api/aaa/user/${this.userId}`).then((response) => {
        this.viewableUser = response.data;
      })
    },
    tetATet(withUserId) {
      axios.put(`/api/chat/tet-a-tet/${withUserId}`).then(response => {
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
    },
    onProfileSet() {
      this.loadUser();
      this.graphQlUserStatusSubscribe();
    },
    canDrawUsers() {
      return !!this.chatStore.currentUser
    },
  },
  mounted() {
    bus.on(LOGGED_OUT, this.onLoggedOut);
    bus.on(PROFILE_SET, this.onProfileSet);
    this.setMainTitle()

    if (this.canDrawUsers()) {
      this.onProfileSet();
    }
  },
  beforeUnmount() {
    bus.off(LOGGED_OUT, this.onLoggedOut);
    bus.off(PROFILE_SET, this.onProfileSet);
    this.graphQlUserStatusUnsubscribe();
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
