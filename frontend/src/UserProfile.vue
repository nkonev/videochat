<template>
    <v-card v-if="viewableUser"
            class="mr-auto"
            max-width="640"
    >
        <v-list-item three-line>
            <v-list-item-content class="d-flex justify-space-around">
                <div class="overline mb-4">{{ $vuetify.lang.t('$vuetify.user_profile') }} #{{ viewableUser.id }}</div>
                <v-img v-if="viewableUser.avatarBig || viewableUser.avatar"
                       :src="getAvatar(viewableUser)"
                       :aspect-ratio="16/9"
                       min-width="200"
                       min-height="200"
                >
                </v-img>
                <v-list-item-title class="headline mb-1 mt-2">
                    {{ viewableUser.login }}
                    <template>
                        <span v-if="online" class="grey--text"><v-icon color="success">mdi-checkbox-marked-circle</v-icon> {{ $vuetify.lang.t('$vuetify.user_online') }}</span>
                        <span v-else class="grey--text"><v-icon color="error">mdi-checkbox-marked-circle</v-icon> {{ $vuetify.lang.t('$vuetify.user_offline') }}</span>
                    </template>
                </v-list-item-title>
                <v-list-item-subtitle v-if="viewableUser.email">{{ viewableUser.email }}</v-list-item-subtitle>

                <v-container class="ma-0 pa-0">
                    <v-btn v-if="isNotMyself()" color="primary" @click="tetATet(viewableUser.id)">
                        <v-icon>mdi-message-text-outline</v-icon>
                        {{ $vuetify.lang.t('$vuetify.user_open_chat') }}
                    </v-btn>
                </v-container>
            </v-list-item-content>
        </v-list-item>

        <v-divider class="mx-4"></v-divider>
        <v-card-title class="title pb-0 pt-1">{{ $vuetify.lang.t('$vuetify.bound_oauth2_providers') }}</v-card-title>
        <v-card-actions class="mx-2">
            <v-chip
                v-if="viewableUser.oauth2Identifiers.vkontakteId && providers.includes('vkontakte')"
                min-width="80px"
                label
                class="c-btn-vk py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'vk'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="viewableUser.oauth2Identifiers.facebookId && providers.includes('facebook')"
                min-width="80px"
                label
                class="c-btn-fb py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'facebook'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="viewableUser.oauth2Identifiers.googleId && providers.includes('google')"
                min-width="80px"
                label
                class="c-btn-google py-5 mr-2"
                text-color="white"
            >
                <font-awesome-icon :icon="{ prefix: 'fab', iconName: 'google'}" :size="'2x'"></font-awesome-icon>
            </v-chip>

            <v-chip
                v-if="viewableUser.oauth2Identifiers.keycloakId && providers.includes('keycloak')"
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

import {
    GET_AVAILABLE_OAUTH2_PROVIDERS,
    GET_USER,
    SET_CHAT_ID,
    SET_CHAT_USERS_COUNT,
    SET_SHOW_CHAT_EDIT_BUTTON,
    SET_SHOW_SEARCH,
    SET_TITLE
} from "./store";
import axios from "axios";
import {chat_name} from "./routes";
import {mapGetters} from "vuex";
import graphqlSubscriptionMixin from "@/graphqlSubscriptionMixin";

export default {
    mixins: [graphqlSubscriptionMixin('userOnlineInProfile')],
    data() {
        return {
            viewableUser: null,
            online: false,
        }
    },
    computed: {
        userId() {
            return this.$route.params.id
        },
        ...mapGetters({
            currentUser: GET_USER, // currentUser is here, 'getUser' -- in store.js
            providers: GET_AVAILABLE_OAUTH2_PROVIDERS
        }),
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
            return this.currentUser && this.currentUser.id != this.viewableUser.id
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
            dtos.forEach(dtoItem => {
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
    },
    mounted() {
        this.$store.commit(SET_TITLE, this.$vuetify.lang.t('$vuetify.user_profile'));
        this.$store.commit(SET_CHAT_USERS_COUNT, 0);
        this.$store.commit(SET_SHOW_SEARCH, false);
        this.$store.commit(SET_CHAT_ID, null);
        this.$store.commit(SET_SHOW_CHAT_EDIT_BUTTON, false);

        this.loadUser();
        this.graphQlSubscribe();
    },
    beforeDestroy() {
        this.graphQlUnsubscribe();
    },
    created() {
    },
    destroyed() {
    },
    watch: {
        '$vuetify.lang.current': {
            handler: function (newValue, oldValue) {
                this.$store.commit(SET_TITLE, this.$vuetify.lang.t('$vuetify.user_profile'));
            },
        }
    },
}
</script>

<style lang="stylus">
@import "OAuth2.styl"
</style>
