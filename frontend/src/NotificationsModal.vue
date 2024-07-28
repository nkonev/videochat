<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title class="d-flex align-center ml-2 my-2">
                    {{ $vuetify.locale.t('$vuetify.notifications') }}
                    <v-spacer/>
                    <v-btn v-if="itemsDto.items.length > 0" variant="outlined" @click="onClickClearAllNotifications()"><v-icon>mdi-delete</v-icon> {{ $vuetify.locale.t('$vuetify.clear') }} </v-btn>
                </v-card-title>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0 notification-list" v-if="!loading">
                        <template v-if="itemsDto.items.length > 0">
                            <template v-for="(item, index) in itemsDto.items">
                                <v-list-item link @click.prevent="onNotificationClick(item)" :href="getLink(item)" >
                                  <template v-slot:prepend>
                                    <v-icon size="x-large">
                                      {{ getNotificationIcon(item.notificationType) }}
                                    </v-icon>
                                  </template>
                                  <v-list-item-title>{{ getNotificationTitle(item)}}</v-list-item-title>
                                  <v-list-item-subtitle>{{ getNotificationSubtitle(item) }}</v-list-item-subtitle>
                                  <v-list-item-subtitle>
                                    {{ getNotificationDate(item)}}
                                  </v-list-item-subtitle>
                                </v-list-item>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_notifications') }}</v-card-text>
                        </template>

                    </v-list>

                    <v-progress-circular
                      class="ma-4"
                      v-else
                      indeterminate
                      color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">

                  <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                  <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                    <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0" :class="isMobile() ? 'mb-2' : ''">
                      <v-pagination
                        variant="elevated"
                        active-color="primary"
                        density="comfortable"
                        v-if="shouldShowPagination"
                        v-model="page"
                        :length="pagesCount"
                        :total-visible="getTotalVisible()"
                      ></v-pagination>
                    </v-col>
                    <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-1 flex-shrink-0 align-self-end justify-end">
                      <v-btn variant="outlined" @click="openNotificationSettings()" min-width="0" :title="$vuetify.locale.t('$vuetify.settings')"><v-icon size="large">mdi-cog</v-icon></v-btn>
                      <v-btn
                        variant="elevated"
                        color="red"
                        @click="closeModal()"
                      >
                        {{ $vuetify.locale.t('$vuetify.close') }}
                      </v-btn>
                    </v-col>
                  </v-row>
                </v-card-actions>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    LOGGED_OUT,
    NOTIFICATION_ADD, NOTIFICATION_CLEAR_ALL, NOTIFICATION_DELETE,
    OPEN_NOTIFICATIONS_DIALOG, OPEN_SETTINGS,
} from "./bus/bus";
import {getNotificationSubtitle, getNotificationTitle, hasLength} from "./utils";
import { getHumanReadableDate } from "@/date.js";
import axios from "axios";
import {chat, chat_name, messageIdHashPrefix, videochat_name} from "@/router/routes";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import pageableModalMixin, {pageSize} from "@/mixins/pageableModalMixin.js";

export default {
    mixins: [
        pageableModalMixin()
    ],
    methods: {
        isCachedRelevantToArguments() {
            return true
        },
        initializeWithArguments() {
            // empty
        },
        initiateRequest() {
            return axios.get(`/api/notification/list`, {
                params: {
                    page: this.translatePage(),
                    size: pageSize,
                },
            })
        },

        extractDtoFromEventDto(dto) {
            return [dto.notificationDto]
        },

        initiateFilteredRequest(dto) {
            return Promise.resolve({
                data: [
                    {
                        id: dto.notificationDto.id
                    }
                ]
            })
        },
        initiateCountRequest(dto) {
            return Promise.resolve({
                data: {
                    count: dto.count
                }
            })
        },

        notificationAdd(payload) {
          this.chatStore.setNotificationCount(payload.count);

          this.onItemCreatedEvent(payload);
        },
        notificationDelete(payload) {
          this.chatStore.setNotificationCount(payload.count);

          this.onItemRemovedEvent(payload);
        },
        notificationClearAll() {
            this.chatStore.setNotificationCount(0);
            this.itemsDto = {
                count: 0,
                items: []
            }
        },

        clearOnClose() {
            // empty
        },
        clearOnReset() {
            // empty
        },
        getNotificationIcon(type) {
            switch (type) {
                case "missed_call":
                    return "mdi-phone-missed"
                case "mention":
                    return "mdi-at"
                case "reply":
                  return "mdi-reply-outline"
                case "reaction":
                  return "mdi-emoticon-outline"
            }
        },
        getNotificationSubtitle(item) {
            return getNotificationSubtitle(this.$vuetify, item);
        },
        getNotificationTitle(item) {
            return getNotificationTitle(item)
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
            const routeDto = {name: chat_name, params: {id: item.chatId}};
            if (this.chatId == item.chatId && this.$route.name == videochat_name) {
                routeDto.name = videochat_name; // Improves clicking on 'Missed call' behaviour during the existing call
            }
            if (item.messageId) {
                routeDto.hash = messageIdHashPrefix + item.messageId;
            }
            this.$router.push(routeDto)
                .catch(() => { })
                .then(() => {
                    this.closeModal();
                    axios.put('/api/notification/read/' + item.id);
                })
        },
        openNotificationSettings() {
            bus.emit(OPEN_SETTINGS, 'the_notifications')
        },
        onClickClearAllNotifications() {
            axios.delete('/api/notification')
        },
        resetOnRouteIdChange() {
            return false
        },
        shouldReactOnPageChange() {
            return true
        },
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
    },
    mounted() {
        bus.on(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
        bus.on(NOTIFICATION_ADD, this.notificationAdd);
        bus.on(NOTIFICATION_DELETE, this.notificationDelete);
        bus.on(NOTIFICATION_CLEAR_ALL, this.notificationClearAll);
        bus.on(LOGGED_OUT, this.onLogout);
    },
    beforeUnmount() {
        bus.off(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
        bus.off(NOTIFICATION_ADD, this.notificationAdd);
        bus.off(NOTIFICATION_DELETE, this.notificationDelete);
        bus.off(NOTIFICATION_CLEAR_ALL, this.notificationClearAll);
        bus.off(LOGGED_OUT, this.onLogout);
    },
}
</script>

<style lang="stylus">
.notification-list {
  .v-list-item__prepend {
    margin-right: 1em;
    width: 32px;
  }
}
</style>
