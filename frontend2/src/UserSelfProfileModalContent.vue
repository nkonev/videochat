<template>
        <v-card-title class="title pb-0 pt-2">{{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ chatStore.currentUser.id }}</v-card-title>

        <v-container class="d-flex justify-space-around flex-column py-0 user-self-settings-container">
            <v-img v-if="hasAva"
                   :src="ava"
                   class="mt-2"
            >
            </v-img>

            <v-container class="ma-0 pa-0 mt-2 d-flex flex-row">
              <v-list-item-title v-if="!showLoginInput" class="align-self-center text-h3">{{ chatStore.currentUser.login }}</v-list-item-title>
              <v-btn v-if="!showLoginInput" color="primary" rounded="0" variant="plain" icon :title="$vuetify.locale.t('$vuetify.change_login')" @click="showLoginInput = !showLoginInput; loginPrevious = chatStore.currentUser.login">
                <v-icon dark size="x-large">mdi-lead-pencil</v-icon>
              </v-btn>
              <v-container v-if="showLoginInput" class="ma-0 pa-0 d-flex flex-row">
                <v-text-field
                  v-model="chatStore.currentUser.login"
                  :rules="[rules.required]"
                  :label="$vuetify.locale.t('$vuetify.login')"
                  @keyup.native.enter="sendLogin()"
                  variant="outlined"
                  density="compact"
                  class="mt-3"
                >
                  <template v-slot:append>
                    <v-icon @click="sendLogin()" color="primary" class="mx-1 ml-2">mdi-check-bold</v-icon>
                    <v-icon @click="showLoginInput = false; chatStore.currentUser.login = loginPrevious" class="mx-1">mdi-cancel</v-icon>
                  </template>
                </v-text-field>
              </v-container>
            </v-container>
            <v-divider></v-divider>

            <v-container class="ma-0 pa-0 d-flex flex-row">
              <v-list-item-subtitle v-if="!showEmailInput && chatStore.currentUser.email" class="align-self-center text-h6">{{ chatStore.currentUser.email }}</v-list-item-subtitle>
              <v-btn v-if="!showEmailInput" color="primary" size="x-small" rounded="0" variant="plain" icon :title="$vuetify.locale.t('$vuetify.change_email')" @click="showEmailInput = !showEmailInput; emailPrevious = chatStore.currentUser.email">
                <v-icon dark>mdi-lead-pencil</v-icon>
              </v-btn>
              <v-container v-if="showEmailInput" class="ma-0 pa-0 d-flex flex-row">
                <v-text-field
                  v-model="chatStore.currentUser.email"
                  :rules="[rules.required, rules.email]"
                  label="E-mail"
                  @keyup.native.enter="sendEmail()"
                  variant="outlined"
                  density="compact"
                  class="mt-3"
                >
                  <template v-slot:append>
                    <v-icon @click="sendEmail()" color="primary" class="mx-1 ml-2">mdi-check-bold</v-icon>
                    <v-icon @click="showEmailInput = false; chatStore.currentUser.email = emailPrevious" class="mx-1">mdi-cancel</v-icon>
                  </template>
                </v-text-field>
              </v-container>
            </v-container>

        </v-container>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.bound_oauth2_providers') }}</v-card-title>
        <v-card-actions class="mx-2" v-if="shouldShowBound()">
            <v-chip
                v-if="chatStore.currentUser.oauth2Identifiers.vkontakteId"
                min-width="80px"
                label
                close
                class="c-btn-vk py-5 mr-2"
                text-color="white"
                closable
                close-icon="mdi-delete"
                @click:close="removeVk"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="chatStore.currentUser.oauth2Identifiers.facebookId"
                min-width="80px"
                label
                close
                class="c-btn-fb py-5 mr-2"
                text-color="white"
                closable
                close-icon="mdi-delete"
                @click:close="removeFb"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="chatStore.currentUser.oauth2Identifiers.googleId"
                min-width="80px"
                label
                close
                class="c-btn-google py-5 mr-2"
                text-color="white"
                closable
                close-icon="mdi-delete"
                @click:close="removeGoogle"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="chatStore.currentUser.oauth2Identifiers.keycloakId"
                min-width="80px"
                label
                close
                class="c-btn-keycloak py-5 mr-2"
                text-color="white"
                closable
                close-icon="mdi-delete"
                @click:close="removeKeycloak"
            >
                <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'key'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

        </v-card-actions>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.not_bound_oauth2_providers') }}</v-card-title>
        <v-card-actions class="mx-2" v-if="shouldShowUnbound()">
            <v-chip
                v-if="!chatStore.currentUser.oauth2Identifiers.vkontakteId"
                @click="submitOauthVkontakte"
                min-width="80px"
                label
                class="c-btn-vk py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="!chatStore.currentUser.oauth2Identifiers.facebookId"
                @click="submitOauthFacebook"
                min-width="80px"
                label
                class="c-btn-fb py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="!chatStore.currentUser.oauth2Identifiers.googleId"
                @click="submitOauthGoogle"
                min-width="80px"
                label
                class="c-btn-google py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="!chatStore.currentUser.oauth2Identifiers.keycloakId"
                @click="submitOauthKeycloak"
                min-width="80px"
                label
                class="c-btn-keycloak py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'key'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

        </v-card-actions>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.password') }}</v-card-title>
        <v-btn v-if="!showPasswordInput" class="mx-4 mb-4" color="primary" dark
               @click="showPasswordInput = !showPasswordInput">
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.change_password') }}
            </template>
            <template v-slot:append>
              <v-icon dark>mdi-lock</v-icon>
            </template>
        </v-btn>
        <v-container v-if="showPasswordInput" class="ma-0 py-0 d-flex flex-row user-self-settings-container">
          <v-text-field
            v-model="password"
            :type="showInputablePassword ? 'text' : 'password'"
            :rules="[rules.required, rules.min]"
            :label="$vuetify.locale.t('$vuetify.password')"
            @keyup.native.enter="sendPassword()"
            variant="outlined"
            density="compact"
          >
            <template v-slot:append>
              <v-icon @click="showInputablePassword = !showInputablePassword" class="mx-1 ml-3">{{showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'}}</v-icon>
              <v-icon @click="sendPassword()" color="primary" class="mx-1">mdi-check-bold</v-icon>
              <v-icon @click="showPasswordInput = false" class="mx-1">mdi-cancel</v-icon>
            </template>
          </v-text-field>
        </v-container>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.short_info') }}</v-card-title>
        <v-btn v-if="!showShortInfoInput" class="mx-4 mb-4" color="primary" dark
               @click="showShortInfoInput = !showShortInfoInput; shortInfoPrevious = chatStore.currentUser.shortInfo">
            <template v-slot:default>
              {{ $vuetify.locale.t('$vuetify.change_short_info') }}
            </template>
            <template v-slot:append>
              <v-icon dark>mdi-information</v-icon>
            </template>
        </v-btn>
        <v-container v-if="showShortInfoInput" class="ma-0 py-0 d-flex flex-row user-self-settings-container">
          <v-text-field
            v-model="chatStore.currentUser.shortInfo"
            label="Short info"
            @keyup.native.enter="sendShortInfo()"
            variant="outlined"
            density="compact"
            hide-details
          >
            <template v-slot:append>
              <v-icon @click="sendShortInfo()" color="primary" class="mx-1 ml-3">mdi-check-bold</v-icon>
              <v-icon @click="showShortInfoInput = false; chatStore.currentUser.shortInfo = shortInfoPrevious" class="mx-1">mdi-cancel</v-icon>
            </template>
          </v-text-field>
        </v-container>
</template>

<script>
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {deepCopy, hasLength, setTitle} from "@/utils";

export default {
    props: ['enabled'],
    data() {
        return {
            showInputablePassword: false,

            showLoginInput: false,
            showPasswordInput: false,
            showEmailInput: false,
            showShortInfoInput: false,

            loginPrevious: "",
            password: "",
            emailPrevious: "",
            shortInfoPrevious: null,
        }
    },
    computed: {
        ...mapStores(useChatStore),
        ava() {
            const maybeUser = this.chatStore.currentUser;
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
          const maybeUser = this.chatStore.currentUser;
          return hasLength(maybeUser?.avatarBig) || hasLength(maybeUser?.avatar)
        },
        rules() {
          const minChars = 8;
          const requiredMessage = this.$vuetify.locale.t('$vuetify.required');
          const minCharsMessage = this.$vuetify.locale.t('$vuetify.min_characters', minChars);
          const invalidEmailMessage = this.$vuetify.locale.t('$vuetify.invalid_email');
          return {
            required: value => !!value || requiredMessage,
            min: v => v.length >= minChars || minCharsMessage,
            email: value => {
              const pattern = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
              return pattern.test(value) || invalidEmailMessage
            },
          }
        },

    },
    methods: {
        shouldShowBound() {
          const copied = deepCopy(this.chatStore.currentUser.oauth2Identifiers);
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
        shouldShowUnbound() {
          return this.chatStore.availableOAuth2Providers.length && (
            this.chatStore.availableOAuth2Providers.includes('vkontakte') ||
            this.chatStore.availableOAuth2Providers.includes('facebook') ||
            this.chatStore.availableOAuth2Providers.includes('google') ||
            this.chatStore.availableOAuth2Providers.includes('keycloak')
          )
        },
        submitOauthVkontakte() {
            window.location.href = '/api/login/oauth2/vkontakte';
        },
        submitOauthFacebook() {
            window.location.href = '/api/login/oauth2/facebook';
        },
        submitOauthGoogle() {
            window.location.href = '/api/login/oauth2/google';
        },
        submitOauthKeycloak() {
            window.location.href = '/api/login/oauth2/keycloak';
        },

        sendLogin() {
            axios.patch('/api/profile', {login: this.chatStore.currentUser.login})
                .then((response) => {
                    this.chatStore.fetchUserProfile()
                    this.showLoginInput = false;
                })
        },
        sendPassword() {
            axios.patch('/api/profile', {password: this.password})
                .then((response) => {
                    this.showPasswordInput = false;
                })
        },
        sendEmail() {
            axios.patch('/api/profile', {email: this.chatStore.currentUser.email})
                .then((response) => {
                    this.chatStore.fetchUserProfile()
                    this.showEmailInput = false;
                })
        },
        sendShortInfo() {
            axios.patch('/api/profile', {shortInfo: this.chatStore.currentUser.shortInfo})
                .then((response) => {
                    this.chatStore.fetchUserProfile()
                    this.showShortInfoInput = false;
                })
        },
        removeVk() {
            axios.delete('/api/profile/vkontakte')
                .then((response) => {
                  this.chatStore.fetchUserProfile()
                })
        },
        removeFb() {
            axios.delete('/api/profile/facebook')
                .then((response) => {
                  this.chatStore.fetchUserProfile()
                })
        },
        removeGoogle() {
            axios.delete('/api/profile/google')
                .then((response) => {
                  this.chatStore.fetchUserProfile()
                })
        },
        removeKeycloak() {
            axios.delete('/api/profile/keycloak')
                .then((response) => {
                    this.chatStore.fetchUserProfile()
                })
        },
    },
    mounted() {
    },
    beforeUnmount() {
    },
}
</script>

<style lang="stylus" scoped>
  @import "oAuth2.styl"
</style>

<style lang="stylus">
.user-self-settings-container {
  .v-input__append {
    margin-inline-start: unset
  }
}
</style>
