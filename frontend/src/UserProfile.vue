<template>
    <v-card v-if="viewableUser"
            class="mr-auto"
            max-width="640"
    >
        <v-list-item three-line>
            <v-list-item-content class="d-flex justify-space-around">
                <div class="overline mb-4">User profile #{{ viewableUser.id }}</div>
                <v-img v-if="viewableUser.avatarBig || viewableUser.avatar"
                       :src="getAvatar(viewableUser)"
                       :aspect-ratio="16/9"
                       min-width="200"
                       min-height="200"
                >
                </v-img>
                <v-list-item-title class="headline mb-1 mt-2">{{ viewableUser.login }} <v-btn v-if="currentUser.id != viewableUser.id" color="primary" icon @click="tetATet(viewableUser.id)"><v-icon>mdi-message-text-outline</v-icon></v-btn></v-list-item-title>
                <v-list-item-subtitle v-if="viewableUser.email">{{ viewableUser.email }}</v-list-item-subtitle>
            </v-list-item-content>
        </v-list-item>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">Bound OAuth2 providers</v-card-title>
        <v-card-actions class="mx-2">
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
        </v-card-actions>
    </v-card>
</template>

<script>

import {
    GET_USER,
    SET_CHAT_ID,
    SET_CHAT_USERS_COUNT,
    SET_SHOW_CHAT_EDIT_BUTTON,
    SET_SHOW_SEARCH,
    SET_TITLE
} from "./store";
import axios from "axios";
import {getCorrectUserAvatar} from "./utils";
import {chat_name} from "./routes";
import {mapGetters} from "vuex";

export default {
    data() {
        return {
            viewableUser: null
        }
    },
    computed: {
        userId() {
            return this.$route.params.id
        },
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
    },
    methods: {
        getAvatar(u) {
            if (u.avatarBig) {
                return getCorrectUserAvatar(u.avatarBig)
            } else if (u.avatar) {
                return getCorrectUserAvatar(u.avatar)
            } else {
                return null
            }
        },
        loadUser() {
            this.viewableUser = null;
            axios.get('/api/user/list', {
                params: {userId: this.userId}
            }).then((response) => {
                if (response.data.length) {
                    this.viewableUser = response.data[0];
                }
            })
        },
        tetATet(withUserId) {
            axios.put(`/api/chat/tet-a-tet/${withUserId}`).then(response => {
                this.$router.push(({ name: chat_name, params: { id: response.data.id}}));
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