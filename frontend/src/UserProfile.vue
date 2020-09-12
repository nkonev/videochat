<template>
    <v-card v-if="currentUser"
            class="mr-auto"
            max-width="640"
    >
      <v-list-item three-line>
        <v-list-item-content class="d-flex justify-space-around">
          <div class="overline mb-4">User profile</div>
          <v-img  v-if="currentUser.avatar"
                  :src="currentUser.avatar"
                  :aspect-ratio="16/9"
                  min-width="200"
                  min-height="200"
          >
          </v-img>
          <v-list-item-title class="headline mb-1 mt-2">{{ currentUser.login }}</v-list-item-title>
          <v-list-item-subtitle v-if="currentUser.email">{{currentUser.email}}</v-list-item-subtitle>
        </v-list-item-content>
      </v-list-item>

      <v-divider class="mx-4"></v-divider>
      <v-card-title class="title pb-0 pt-1">Bound OAuth2 providers</v-card-title>
      <v-card-actions class="mx-2">
          <v-card v-if="currentUser.oauthIdentifiers.vkontakteId" min-width="80px" class="text-center pa-1 d-flex justify-center mr-2 c-btn-vk"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon></v-card>
          <v-card v-if="currentUser.oauthIdentifiers.facebookId" min-width="80px" class="text-center pa-1 d-flex justify-center mr-2 c-btn-fb"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon></v-card>
      </v-card-actions>

      <v-divider class="mx-4"></v-divider>
      <v-card-title class="title pb-0 pt-1">Not bound OAuth2 providers</v-card-title>
      <v-card-actions class="mx-2">
        <v-btn v-if="!currentUser.oauthIdentifiers.vkontakteId" class="mr-2 c-btn-vk" min-width="80px"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon></v-btn>
        <v-btn v-if="!currentUser.oauthIdentifiers.facebookId" class="mr-2 c-btn-fb" min-width="80px"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook' }" :size="'2x'"></font-awesome-icon></v-btn>
      </v-card-actions>

      <v-divider class="mx-4"></v-divider>
      <v-card-title class="title pb-0 pt-1">Login</v-card-title>
      <v-btn class="mx-4 mb-4" color="primary" dark>Change login
        <v-icon dark right>mdi-account</v-icon>
      </v-btn>
      <v-text-field class="mx-4"
                    label="Login"
                    append-outer-icon="mdi-check-bold"
                    :rules="[rules.required]"
                    @click:append-outer="sendLogin"
                    v-model="login"></v-text-field>

      <v-divider class="mx-4"></v-divider>
      <v-card-title class="title pb-0 pt-1">Password</v-card-title>
      <v-btn class="mx-4 mb-4" color="primary" dark>Change password
        <v-icon dark right>mdi-lock</v-icon>
      </v-btn>
      <v-text-field
          class="mx-4"
          v-model="password"
          append-outer-icon="mdi-check-bold"
          @click:append-outer="sendPassword"
          :append-icon="showPassword ? 'mdi-eye' : 'mdi-eye-off'"
          :type="showPassword ? 'text' : 'password'"
          :rules="[rules.required, rules.min]"
          label="Password"
          hint="At least 8 characters"
          @click:append="showPassword = !showPassword"
      ></v-text-field>

      <v-divider class="mx-4"></v-divider>
      <v-card-title class="title pb-0 pt-1">Email</v-card-title>
      <v-btn class="mx-4 mb-4" color="primary" dark>Change email
        <v-icon dark right>mdi-email</v-icon>
      </v-btn>
      <v-text-field
          class="mx-4"
          v-model="email"
          append-outer-icon="mdi-check-bold"
          @click:append-outer="sendEmail"
          :rules="[rules.required, rules.email]"
          label="E-mail"
      ></v-text-field>

    </v-card>
</template>

<script>
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";

    export default {
        data() {
          return {
            showPassword: false,
            rules: {
              required: value => !!value || 'Required.',
              min: v => v.length >= 8 || 'Min 8 characters',
              email: value => {
                const pattern = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
                return pattern.test(value) || 'Invalid e-mail.'
              },
            },

            login: "",
            password: "",
            email: ""
          }
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
        methods: {
          sendLogin() {

          },
          sendPassword() {

          },
          sendEmail() {

          }
        }
    }
</script>

<style lang="stylus">
    @import "OAuth2.styl"
</style>