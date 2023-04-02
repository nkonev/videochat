<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="720" scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.notifications') }}</v-card-title>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0">
                        <template v-if="notifications.length > 0">
                            <template v-for="(item, index) in notifications">
                                <v-list-item link @click.prevent="onNotificationClick(item)" :href="getLink(item)">
                                    <v-list-item-icon class="mr-4"><v-icon large>{{getNotificationIcon(item.notificationType)}}</v-icon></v-list-item-icon>
                                    <v-list-item-content class="py-2">
                                        <v-list-item-title>{{ getNotificationTitle(item)}}</v-list-item-title>
                                        <v-list-item-subtitle>{{ getNotificationSubtitle(item) }}</v-list-item-subtitle>
                                        <v-list-item-subtitle>
                                            {{ getNotificationDate(item)}}
                                        </v-list-item-subtitle>
                                    </v-list-item-content>
                                </v-list-item>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_notifications') }}</v-card-text>
                        </template>

                    </v-list>
                </v-card-text>
                <v-divider/>
                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_mentions')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4 py-1"
                        v-model="notificationsSettings.mentionsEnabled"
                        @click="putNotificationsSettings()"
                    ></v-switch>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_missed_calls')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4 py-1"
                        v-model="notificationsSettings.missedCallsEnabled"
                        @click="putNotificationsSettings()"
                    ></v-switch>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_replies')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4 py-1"
                        v-model="notificationsSettings.answersEnabled"
                        @click="putNotificationsSettings()"
                    ></v-switch>

                    <v-spacer></v-spacer>

                    <v-btn
                        color="error"
                        class="my-1"
                        @click="closeModal()"
                    >
                        {{ $vuetify.lang.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_NOTIFICATIONS_DIALOG,
} from "./bus";
import {mapGetters} from 'vuex'
import {GET_NOTIFICATIONS, GET_NOTIFICATIONS_SETTINGS, SET_NOTIFICATIONS_SETTINGS} from "@/store";
import {getHumanReadableDate} from "./utils";
import axios from "axios";
import {chat, chat_name, messageIdHashPrefix} from "@/routes";

export default {
    data () {
        return {
            show: false,
        }
    },

    methods: {
        showModal() {
            this.show = true;
        },
        closeModal() {
            this.show = false;
        },
        getNotificationIcon(type) {
            switch (type) {
                case "missed_call":
                    return "mdi-phone-missed"
                case "mention":
                    return "mdi-at"
                case "reply":
                    return "mdi-reply-outline"
            }
        },
        getNotificationSubtitle(item) {
            switch (item.notificationType) {
                case "missed_call":
                    return this.$vuetify.lang.t('$vuetify.notification_missed_call', item.byLogin)
                case "mention":
                    return this.$vuetify.lang.t('$vuetify.notification_mention', item.byLogin)
                case "reply":
                    return this.$vuetify.lang.t('$vuetify.notification_reply', item.byLogin)
            }
        },
        getNotificationTitle(item) {
            return item.description
        },
        getNotificationDate(item) {
            return getHumanReadableDate(item.createDateTime)
        },
        getLink(item) {
            let url = chat + "/" + item.chatId;
            if (item.messageId) {
                url += messageIdHashPrefix + item.messageId;
            }
            return url;
        },
        onNotificationClick(item) {
            if (this.chatId != item.chatId) {
                const routeDto = {name: chat_name, params: {id: item.chatId}};
                if (item.messageId) {
                    routeDto.hash = messageIdHashPrefix + item.messageId;
                }
                this.$router.push(routeDto)
                    .catch(() => { })
                    .then(() => {
                        this.closeModal();
                        axios.put('/api/notification/read/' + item.id);
                    })
            } else {
                this.closeModal();
                axios.put('/api/notification/read/' + item.id);
            }
        },
        putNotificationsSettings() {
            axios.put('/api/notification/settings', this.notificationsSettings).then(({data}) => {
                this.$store.commit(SET_NOTIFICATIONS_SETTINGS, data);
            })
        },
    },
    computed: {
        ...mapGetters({
            notifications: GET_NOTIFICATIONS,
            notificationsSettings: GET_NOTIFICATIONS_SETTINGS,
        }),
        chatId() {
            return this.$route.params.id
        },
    },
    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        }
    },
    created() {
        bus.$on(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
    },
}
</script>
