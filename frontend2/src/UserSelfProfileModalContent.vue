<template>
        <v-card-title class="title pb-0 pt-2">{{ $vuetify.locale.t('$vuetify.user_profile') }} #{{ chatStore.currentUser.id }}</v-card-title>

        <v-container class="d-flex justify-space-around flex-column">
            <v-img v-if="chatStore.currentUser.avatarBig || chatStore.currentUser.avatar"
                   :src="ava"
                   :aspect-ratio="16/9"
                   min-width="400"
                   min-height="400"
                   @click="openAvatarDialog"
            >
            </v-img>
            <v-btn v-else color="primary" @click="openAvatarDialog()">{{ $vuetify.locale.t('$vuetify.choose_avatar_btn') }}</v-btn>

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
                  hide-details
                  density="compact"
                  class="mr-1"
                ></v-text-field>
                <v-icon @click="sendLogin()" color="primary" class="mx-1 align-self-center">mdi-check-bold</v-icon>
                <v-icon @click="showLoginInput = false; chatStore.currentUser.login = loginPrevious" class="mx-1 align-self-center">mdi-cancel</v-icon>
              </v-container>
            </v-container>

            <v-list-item-subtitle v-if="chatStore.currentUser.email">{{ chatStore.currentUser.email }}</v-list-item-subtitle>
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
            {{ $vuetify.locale.t('$vuetify.change_password') }}
            <v-icon dark right>mdi-lock</v-icon>
        </v-btn>
        <v-row v-if="showPasswordInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="password"
                            :append-icon="showInputablePassword ? 'mdi-eye' : 'mdi-eye-off'"
                            @click:append="showInputablePassword = !showInputablePassword"
                            :type="showInputablePassword ? 'text' : 'password'"
                            :rules="[rules.required, rules.min]"
                            :label="$vuetify.locale.t('$vuetify.password')"
                            @keyup.native.enter="sendPassword()"
                            variant="outlined"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendPassword()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showPasswordInput = false">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.email') }}</v-card-title>
        <v-btn v-if="!showEmailInput" class="mx-4 mb-4" color="primary" dark @click="showEmailInput = !showEmailInput; emailPrevious = chatStore.currentUser.email">
            {{ $vuetify.locale.t('$vuetify.change_email') }}
            <v-icon dark right>mdi-email</v-icon>
        </v-btn>
        <v-row v-if="showEmailInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="chatStore.currentUser.email"
                            :rules="[rules.required, rules.email]"
                            label="E-mail"
                            @keyup.native.enter="sendEmail()"
                            variant="outlined"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendEmail()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showEmailInput = false; chatStore.currentUser.email = emailPrevious">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>


        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.locale.t('$vuetify.short_info') }}</v-card-title>
        <v-btn v-if="!showShortInfoInput" class="mx-4 mb-4" color="primary" dark @click="showShortInfoInput = !showShortInfoInput; shortInfoPrevious = chatStore.currentUser.shortInfo">
            {{ $vuetify.locale.t('$vuetify.change_short_info') }}
            <v-icon dark right>mdi-information</v-icon>
        </v-btn>
        <v-row v-if="showShortInfoInput" no-gutters>
            <v-col cols="12" >
                <v-row :align="'center'" no-gutters>
                    <v-col class="ml-4">
                        <v-text-field
                            v-model="chatStore.currentUser.shortInfo"
                            label="Short info"
                            @keyup.native.enter="sendShortInfo()"
                            variant="outlined"
                        ></v-text-field>
                    </v-col>
                    <v-col md="auto" class="ml-1 mr-4">
                        <v-row :align="'center'" no-gutters>
                            <v-icon @click="sendShortInfo()" color="primary">mdi-check-bold</v-icon>
                            <v-icon @click="showShortInfoInput = false; chatStore.currentUser.shortInfo = shortInfoPrevious">mdi-cancel</v-icon>
                        </v-row>
                    </v-col>
                </v-row>
            </v-col>
        </v-row>

</template>

<script>
import axios from "axios";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {deepCopy, hasLength, setTitle} from "@/utils";

export default {
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
            shortInfoPrevious: null
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

        openAvatarDialog() {
          console.warn("Not implemented");
            // bus.$emit(OPEN_CHOOSE_AVATAR, {
            //     initialAvatarCallback: () => {
            //         return this.ava
            //     },
            //     uploadAvatarFileCallback: (blob) => {
            //         if (!blob) {
            //             return Promise.resolve(false);
            //         }
            //         const config = {
            //             headers: { 'content-type': 'multipart/form-data' }
            //         }
            //         const formData = new FormData();
            //         formData.append('data', blob);
            //         return axios.post('/api/storage/avatar', formData, config)
            //     },
            //     removeAvatarUrlCallback: () => {
            //         return axios.patch(`/api/profile`, {removeAvatar: true});
            //     },
            //     storeAvatarUrlCallback: (res) => {
            //         return axios.patch(`/api/profile`, {avatar: res.data.relativeUrl, avatarBig: res.data.relativeBigUrl});
            //     },
            //     onSuccessCallback: () => {
            //         this.$store.dispatch(FETCH_USER_PROFILE);
            //     }
            // });
        },
    },
    mounted() {
      this.setMainTitle();
    },
    beforeUnmount() {
      this.unsetMainTitle();
    },
    watch: {
      '$vuetify.locale.current': {
        handler: function (newValue, oldValue) {
          this.setMainTitle();
        },
      }
    },

}
</script>

<style lang="stylus" scoped>
  @import "oAuth2.styl"
</style>
