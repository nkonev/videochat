<template>
    <v-progress-linear
        :active="loading"
        :indeterminate="loading"
        absolute
        bottom
        color="primary"
    ></v-progress-linear>
    <v-card-title :disabled="loading" class="title">{{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ chatStore.currentUser.id }}</v-card-title>
    <v-card :disabled="loading">
        <v-container class="d-flex justify-space-around flex-column py-0 user-self-settings-container">
            <v-img v-if="hasAva"
                   :src="ava"
                   class="mt-2"
            >
            </v-img>

            <v-container class="ma-0 pa-0 mt-2 d-flex flex-row">
              <span v-if="!showLoginInput" class="align-self-center text-h3" :style="getLoginColoredStyle(chatStore.currentUser)">{{ chatStore.currentUser.login }}</span>
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
                  class="mt-3 mb-2"
                  hide-details
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
              <span v-if="!showEmailInput && chatStore.currentUser.email" class="align-self-center text-h6">{{ chatStore.currentUser.email }}</span>
              <v-btn v-if="!showEmailInput" color="primary" size="x-small" rounded="0" variant="plain" icon :title="$vuetify.locale.t('$vuetify.change_email')" @click="showEmailInput = !showEmailInput; emailPrevious = chatStore.currentUser.email">
                <v-icon dark>mdi-lead-pencil</v-icon>
              </v-btn>
              <v-container v-if="showEmailInput" class="ma-0 pa-0 d-flex flex-row">
                <v-text-field
                  v-model="chatStore.currentUser.email"
                  :rules="[rules.required, rules.email]"
                  :label="$vuetify.locale.t('$vuetify.email')"
                  @keyup.native.enter="sendEmail()"
                  variant="outlined"
                  density="compact"
                  class="mt-3 mb-2"
                  hide-details
                >
                  <template v-slot:append>
                    <v-icon @click="sendEmail(); chatStore.currentUser.email = emailPrevious" color="primary" class="mx-1 ml-2">mdi-check-bold</v-icon>
                    <v-icon @click="showEmailInput = false; chatStore.currentUser.email = emailPrevious" class="mx-1">mdi-cancel</v-icon>
                  </template>
                </v-text-field>
              </v-container>
            </v-container>
            <v-container class="ma-0 pa-0 d-flex flex-column text-caption" v-if="chatStore.currentUser.awaitingForConfirmEmailChange">
                <span>{{ $vuetify.locale.t('$vuetify.confirm_email_to_change_role_part_1') }}</span>
                <span>
                    <span>{{ $vuetify.locale.t('$vuetify.confirm_email_to_change_role_part_2') }}</span>
                    <v-btn class="mx-2 mb-1" density="compact" variant="outlined" size="" @click="resendEmailConfirmation()">{{ $vuetify.locale.t('$vuetify.confirm_email_to_change_role_btn') }}</v-btn>
                </span>
            </v-container>

        </v-container>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.bound_oauth2_providers') }}</v-card-title>
        <v-card-actions class="mx-2" v-if="shouldShowBound()">
            <v-chip
                v-if="shouldShowBoundVkontakte()"
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
                v-if="shouldShowBoundFacebook()"
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
                v-if="shouldShowBoundGoogle()"
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
                v-if="shouldShowBoundKeycloak()"
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
                v-if="shouldShowUnboundVkontakte()"
                @click="submitOauthVkontakte"
                min-width="80px"
                label
                class="c-btn-vk py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="shouldShowUnboundFacebook()"
                @click="submitOauthFacebook"
                min-width="80px"
                label
                class="c-btn-fb py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="shouldShowUnboundGoogle()"
                @click="submitOauthGoogle"
                min-width="80px"
                label
                class="c-btn-google py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="shouldShowUnboundKeycloak()"
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

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.login_color') }}</v-card-title>
        <v-btn class="mx-4 mb-4" color="primary" dark
               @click="changeLoginColor()">
            <template v-slot:default>
                {{ $vuetify.locale.t('$vuetify.change_login_color') }}
            </template>
            <template v-slot:append>
                <v-icon dark>mdi-invert-colors</v-icon>
            </template>
        </v-btn>

    </v-card>
</template>

<script>
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {colorLogin, getLoginColoredStyle, hasLength} from "@/utils";
import userProfileValidationRules from "@/mixins/userProfileValidationRules";
import bus, {COLOR_SET, OPEN_CHOOSE_COLOR} from "@/bus/bus";

export default {
    mixins: [userProfileValidationRules()],
    data() {
        return {
            loading: false,

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
          const maybeUser = this.chatStore.currentUser;
          return hasLength(maybeUser?.avatarBig) || hasLength(maybeUser?.avatar)
        },

    },
    methods: {
        getLoginColoredStyle,
        shouldShowBound() {
            return this.shouldShowBoundVkontakte() ||
                this.shouldShowBoundFacebook() ||
                this.shouldShowBoundGoogle() ||
                this.shouldShowBoundKeycloak()
        },

        shouldShowBoundVkontakte() {
            return this.chatStore.currentUser.oauth2Identifiers.vkontakteId &&
                this.chatStore.availableOAuth2Providers.includes('vkontakte')
        },
        shouldShowBoundFacebook() {
            return this.chatStore.currentUser.oauth2Identifiers.facebookId &&
                this.chatStore.availableOAuth2Providers.includes('facebook')
        },
        shouldShowBoundGoogle() {
            return this.chatStore.currentUser.oauth2Identifiers.googleId &&
                this.chatStore.availableOAuth2Providers.includes('google')
        },
        shouldShowBoundKeycloak() {
            return this.chatStore.currentUser.oauth2Identifiers.keycloakId &&
                this.chatStore.availableOAuth2Providers.includes('keycloak')
        },



        shouldShowUnbound() {
            return this.shouldShowUnboundVkontakte() ||
                    this.shouldShowUnboundFacebook() ||
                    this.shouldShowUnboundGoogle() ||
                    this.shouldShowUnboundKeycloak()
        },
        shouldShowUnboundVkontakte() {
            return !this.chatStore.currentUser.oauth2Identifiers.vkontakteId &&
              this.chatStore.availableOAuth2Providers.includes('vkontakte')
        },
        shouldShowUnboundFacebook() {
            return !this.chatStore.currentUser.oauth2Identifiers.facebookId &&
                this.chatStore.availableOAuth2Providers.includes('facebook')
        },
        shouldShowUnboundGoogle() {
            return !this.chatStore.currentUser.oauth2Identifiers.googleId &&
                this.chatStore.availableOAuth2Providers.includes('google')
        },
        shouldShowUnboundKeycloak() {
            return !this.chatStore.currentUser.oauth2Identifiers.keycloakId &&
                this.chatStore.availableOAuth2Providers.includes('keycloak')
        },

        submitOauthVkontakte() {
            this.loading = true;
            window.location.href = '/api/aaa/login/oauth2/vkontakte';
        },
        submitOauthFacebook() {
            this.loading = true;
            window.location.href = '/api/aaa/login/oauth2/facebook';
        },
        submitOauthGoogle() {
            this.loading = true;
            window.location.href = '/api/aaa/login/oauth2/google';
        },
        submitOauthKeycloak() {
            this.loading = true;
            window.location.href = '/api/aaa/login/oauth2/keycloak';
        },

        sendLogin() {
            this.loading = true;
            axios.patch('/api/aaa/profile', {login: this.chatStore.currentUser.login})
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                    this.showLoginInput = false;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        sendPassword() {
            this.loading = true;
            axios.patch('/api/aaa/profile', {password: this.password})
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                    this.showPasswordInput = false;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        sendEmail() {
            this.loading = true;
            axios.patch('/api/aaa/profile', {email: this.chatStore.currentUser.email}, { params: {
                language: this.$vuetify.locale.current
            }})
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                    this.showEmailInput = false;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        resendEmailConfirmation() {
            this.loading = true;
            axios.post('/api/aaa/change-email/resend', null, { params: {
                    language: this.$vuetify.locale.current
                }}).finally(()=>{
                    this.loading = false;
                })
        },
        sendShortInfo() {
            this.loading = true;
            axios.patch('/api/aaa/profile', {shortInfo: this.chatStore.currentUser.shortInfo})
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                    this.showShortInfoInput = false;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        removeVk() {
            this.loading = true;
            axios.delete('/api/aaa/profile/vkontakte')
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        removeFb() {
            this.loading = true;
            axios.delete('/api/aaa/profile/facebook')
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        removeGoogle() {
            this.loading = true;
            axios.delete('/api/aaa/profile/google')
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        removeKeycloak() {
            this.loading = true;
            axios.delete('/api/aaa/profile/keycloak')
                .then((response) => {
                    this.chatStore.currentUser = response.data;
                }).finally(()=>{
                    this.loading = false;
                })
        },
        changeLoginColor() {
            bus.emit(OPEN_CHOOSE_COLOR, {colorMode: colorLogin});
        },
        onColorSet({color, colorMode}) {
            if (colorMode == colorLogin) {
                console.debug("Setting color", color, colorMode);
                this.loading = true;
                if (color) {
                    axios.patch('/api/aaa/profile', {loginColor: color})
                        .then((response) => {
                            this.chatStore.currentUser = response.data;
                        }).finally(()=>{
                            this.loading = false;
                        })
                } else {
                    axios.patch('/api/aaa/profile', {loginColor: null, removeLoginColor: true})
                        .then((response) => {
                            this.chatStore.currentUser = response.data;
                        }).finally(()=>{
                            this.loading = false;
                        })
                }
            }
        },
    },
    mounted() {
        bus.on(COLOR_SET, this.onColorSet);
    },
    beforeUnmount() {
        bus.off(COLOR_SET, this.onColorSet);
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
