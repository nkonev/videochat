<template>
    <v-card v-if="currentUser"
            class="mx-auto"
    >
        <v-img  v-if="currentUser.avatar"
                :src="currentUser.avatar"
                max-width="400px"
                max-height="400px"
        ></v-img>

        <v-card-title>
            {{currentUser.login}}
        </v-card-title>

        <v-card-actions>
            <v-btn v-if="currentUser.oauthIdentifiers.vkontakteId" class="mr-2 c-btn-vk" min-width="80px"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon></v-btn>
            <v-btn v-if="currentUser.oauthIdentifiers.facebookId" class="mr-2 c-btn-fb" min-width="80px"><font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook' }" :size="'2x'"></font-awesome-icon></v-btn>
        </v-card-actions>
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