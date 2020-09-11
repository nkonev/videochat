<template>
    <v-card v-if="currentUser"
            class="mr-auto"
            max-width="600"
    >
      <v-list-item three-line>
        <v-list-item-content>
          <div class="overline mb-4">User profile</div>
          <v-img  v-if="currentUser.avatar"
                  :src="currentUser.avatar"
                  max-width="400px"
                  max-height="400px"
          ></v-img>

          <v-list-item-title class="headline mb-1">{{ currentUser.login }}</v-list-item-title>
          <v-list-item-subtitle v-if="currentUser.about">{{currentUser.about}}</v-list-item-subtitle>
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
      <v-card-title class="title pb-0 pt-1">Password</v-card-title>
      <v-btn class="mx-4 mb-4" color="primary" dark>Change password
        <v-icon dark right>mdi-lock</v-icon>
      </v-btn>
    </v-card>
</template>

<script>
    import {mapGetters} from "vuex";
    import {GET_USER} from "./store";

    export default {
        computed: {
            ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        },
    }
</script>

<style lang="stylus">
    @import "OAuth2.styl"
</style>