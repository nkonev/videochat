<template>
  <v-card v-if="viewableUser"
          class="mr-auto"
          max-width="800"
  >
    <v-container class="d-flex justify-space-around flex-column py-0 user-self-settings-container">
      <div class="title pb-0 pt-2">{{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ viewableUser.id }}</div>

      <v-img v-if="hasAva"
         :src="ava"
             max-width="320"
         class="mt-2"
      >
      </v-img>
      <v-list-item-title class="d-flex">
        <span class="text-h3">
          {{ viewableUser.login }}
        </span>

        <span class="ml-2 mb-2 d-flex flex-row align-self-end">
          <span v-if="online" class="text-grey "><v-icon color="success">mdi-checkbox-marked-circle</v-icon> {{ $vuetify.locale.t('$vuetify.user_online') }}</span>
          <span v-else class="text-grey align-self-center"><v-icon color="red">mdi-checkbox-marked-circle</v-icon> {{ $vuetify.locale.t('$vuetify.user_offline') }}</span>
        </span>
      </v-list-item-title>
      <v-divider></v-divider>
      <v-list-item-subtitle v-if="viewableUser.email" class="text-h6">{{ viewableUser.email }}</v-list-item-subtitle>
      <v-divider></v-divider>
      <v-list-item-subtitle v-if="displayShortInfo(viewableUser)" class="my-1">{{ viewableUser.shortInfo }}</v-list-item-subtitle>

      <v-container class="ma-0 pa-0">
        <v-btn v-if="isNotMyself()" color="primary" @click="tetATet(viewableUser.id)">
          <v-icon>mdi-message-text-outline</v-icon>
          {{ $vuetify.locale.t('$vuetify.user_open_chat') }}
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
import graphqlSubscriptionMixin from "@/mixins/graphqlSubscriptionMixin";
import {deepCopy, hasLength, setTitle} from "@/utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

export default {
  mixins: [graphqlSubscriptionMixin('userOnlineInProfile')],
  data() {
    return {
      viewableUser: null,
      online: false,
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
        if (maybeUser.avatarBig) {
          return maybeUser.avatarBig
        } else if (maybeUser.avatar) {
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
    getAvatar(u) {
      if (u.avatarBig) {
        return u.avatarBig
      } else if (u.avatar) {
        return u.avatar
      } else {
        return null
      }
    },
    isNotMyself() {
      return this.chatStore.currentUser && this.chatStore.currentUser.id != this.viewableUser.id
    },
    loadUser() {
      this.viewableUser = null;
      axios.get(`/api/user/${this.userId}`).then((response) => {
        this.viewableUser = response.data;
      })
    },
    tetATet(withUserId) {
      axios.put(`/api/chat/tet-a-tet/${withUserId}`).then(response => {
        this.$router.push(({ name: chat_name, params: { id: response.data.id}}));
      })
    },

    onUserOnlineChanged(rawData) {
      const dtos = rawData?.data?.userOnlineEvents;
      dtos?.forEach(dtoItem => {
        if (dtoItem.id == this.userId) {
          this.online = dtoItem.online;
        }
      })
    },
    getGraphQlSubscriptionQuery() {
      return `
                subscription {
                    userOnlineEvents(userIds:[${this.userId}]) {
                        id
                        online
                    }
                }`
    },
    onNextSubscriptionElement(items) {
      this.onUserOnlineChanged(items);
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
  },
  mounted() {
    this.setMainTitle()

    this.loadUser();
    this.graphQlSubscribe();
  },
  beforeUnmount() {
    this.graphQlUnsubscribe();
    this.unsetMainTitle();
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
