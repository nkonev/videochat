<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.notifications') }}</v-card-title>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0">
                        <template v-if="notifications.length > 0">
                            <template v-for="(item, index) in notifications">
                                <v-list-item link @click="onNotificationClick(item)">
                                    <v-list-item-icon class="mr-4"><v-icon large>{{getNotificationIcon(item.notificationType)}}</v-icon></v-list-item-icon>
                                    <v-list-item-content class="py-2">
                                        <v-list-item-title>{{ getNotificationTitle(item)}}</v-list-item-title>
                                        <v-list-item-subtitle>{{ getNotificationSubtitle(item.notificationType) }}</v-list-item-subtitle>
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
                <v-card-actions>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_mentions')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4"
                        v-model="notificationsSettings.mentionsEnabled"
                        @click="putNotificationsSettings()"
                    ></v-switch>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_missed_calls')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4"
                        v-model="notificationsSettings.missedCallsEnabled"
                        @click="putNotificationsSettings()"
                    ></v-switch>
                    <v-spacer></v-spacer>

                    <v-btn
                        color="error"
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
import { chat_name} from "@/routes";

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
            }
        },
        getNotificationSubtitle(type) {
            switch (type) {
                case "missed_call":
                    return this.$vuetify.lang.t('$vuetify.notification_missed_call')
                case "mention":
                    return this.$vuetify.lang.t('$vuetify.notification_mention')
            }
        },
        getNotificationTitle(item) {
            return item.description
        },
        getNotificationDate(item) {
            return getHumanReadableDate(item.createDateTime)
        },
        onNotificationClick(item) {
            const routeDto = { name: chat_name, params: { id: item.chatId }};
            if (item.messageId) {
                routeDto.hash = require('./routes').messageIdHashPrefix + item.messageId;
            }
            this.$router.push(routeDto).then(()=> {
                this.closeModal();
                axios.put('/api/notification/read/' + item.id);
            })
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
        })
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
