<template>
    <v-card v-if="currentUser"
            class="mr-auto"
            max-width="640"
    >
        <v-list-item three-line>
            <v-list-item-content class="d-flex justify-space-around">
                <div class="overline mb-4">User profile #{{ currentUser.id }}</div>
                <v-img v-if="currentUser.avatar"
                       :src="getAvatar(currentUser.avatar)"
                       :aspect-ratio="16/9"
                       min-width="200"
                       min-height="200"
                >
                </v-img>
                <v-list-item-title class="headline mb-1 mt-2">{{ currentUser.login }}</v-list-item-title>
                <v-list-item-subtitle v-if="currentUser.email">{{ currentUser.email }}</v-list-item-subtitle>
            </v-list-item-content>
        </v-list-item>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Bound OAuth2 providers</v-card-title>
        <v-card-actions class="mx-2">
            <v-chip
                v-if="currentUser.oauth2Identifiers.vkontakteId"
                min-width="80px"
                label
                class="c-btn-vk py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="currentUser.oauth2Identifiers.facebookId"
                min-width="80px"
                label
                class="c-btn-fb py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="currentUser.oauth2Identifiers.googleId"
                min-width="80px"
                label
                class="c-btn-google py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>
        </v-card-actions>
    </v-card>
</template>

<script>

import {
    SET_CHAT_ID,
    SET_CHAT_USERS_COUNT,
    SET_SHOW_CHAT_EDIT_BUTTON,
    SET_SHOW_SEARCH,
    SET_TITLE
} from "./store";
import axios from "axios";
import {getCorrectUserAvatar} from "./utils";

export default {
    data() {
        return {
            currentUser: null
        }
    },
    computed: {
        userId() {
            return this.$route.params.id
        },
    },
    methods: {
        getAvatar(a) {
            return getCorrectUserAvatar(a)
        },
        loadUser() {
            this.currentUser = null;
            axios.get('/api/user/list', {
                params: {userId: this.userId}
            }).then((response) => {
                if (response.data.length) {
                    this.currentUser = response.data[0];
                }
            })

        }
    },
    mounted() {
        this.$store.commit(SET_TITLE, `User profile`);
        this.$store.commit(SET_CHAT_USERS_COUNT, 0);
        this.$store.commit(SET_SHOW_SEARCH, false);
        this.$store.commit(SET_CHAT_ID, null);
        this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);

        this.loadUser();
    },
}
</script>

<style lang="stylus">
@import "OAuth2.styl"
</style>