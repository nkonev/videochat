<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.notifications')">
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0 notification-list" v-if="!loading">
                        <template v-if="dto.data.length > 0">
                            <template v-for="(item, index) in dto.data">
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
  NOTIFICATION_ADD, NOTIFICATION_DELETE,
  OPEN_NOTIFICATIONS_DIALOG, OPEN_SETTINGS,
} from "./bus/bus";
import {findIndex, getHumanReadableDate, hasLength, setIcon} from "./utils";
import axios from "axios";
import {chat, chat_name, messageIdHashPrefix} from "@/router/routes";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

const firstPage = 1;
const pageSize = 20;

const dtoFactory = () => {return {data: [], totalCount: 0} };

export default {
    data () {
        return {
            show: false,
            dto: dtoFactory(),
            page: firstPage,
            loading: false,
        }
    },

    methods: {
        showModal() {
            this.show = true;
            this.loadData();
        },
        loadData() {
          if (!this.show) {
            return
          }
          this.loading = true;
          axios.get(`/api/notification/notification`, {
            params: {
              page: this.translatePage(),
              size: pageSize,
            },
          }).then(({data}) => {
            this.dto = data;
            this.loading = false;
            this.chatStore.notificationsCount = data.totalCount;
            setIcon(data != null && data.totalCount > 0);
          })
        },
        translatePage() {
          return this.page - 1;
        },

        notificationAdd(payload) {
          if (this.show) {
            const newArr = [payload, ...this.dto.data];
            this.dto.data = newArr;
            this.dto.totalCount++;
          }
          this.chatStore.incrementNotificationCount();
        },
        notificationDelete(payload) {
          if (this.show) {
            const arr = this.dto.data;
            const idxToRemove = findIndex(arr, payload);
            arr.splice(idxToRemove, 1);
            this.dto.data = arr;
            this.dto.totalCount--;
          }
          this.chatStore.decrementNotificationCount();
        },

        closeModal() {
            this.show = false;
            this.page = firstPage;
            this.dto = dtoFactory();
            this.loading = false;
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
                    return this.$vuetify.locale.t('$vuetify.notification_missed_call', item.byLogin)
                case "mention":
                    let builder1 = this.$vuetify.locale.t('$vuetify.notification_mention', item.byLogin)
                    if (hasLength(item.chatTitle)) {
                        builder1 += (this.$vuetify.locale.t('$vuetify.in') + "'" + item.chatTitle + "'");
                    }
                    return builder1
                case "reply":
                    let builder2 = this.$vuetify.locale.t('$vuetify.notification_reply', item.byLogin)
                    if (hasLength(item.chatTitle)) {
                        builder2 += (this.$vuetify.locale.t('$vuetify.in') + "'" + item.chatTitle + "'")
                    }
                    return builder2
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
        getTotalVisible() {
            if (!this.isMobile()) {
                return 7
            } else if (this.page == firstPage || this.page == this.pagesCount) {
                return 3
            } else {
                return 1
            }
        },
        openNotificationSettings() {
            bus.emit(OPEN_SETTINGS, 'the_notifications')
        },
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        shouldShowPagination() {
          return this.dto != null && this.dto.data && this.dto.totalCount > pageSize
        },
        pagesCount() {
          const count = Math.ceil(this.dto.totalCount / pageSize);
          // console.debug("Calc pages count", count);
          return count;
        },
    },
    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
        page(newValue) {
          if (this.show) {
            console.debug("SettingNewPage", newValue);
            this.dto = dtoFactory();
            this.loadData();
          }
        },
    },
    mounted() {
        bus.on(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
        bus.on(NOTIFICATION_ADD, this.notificationAdd);
        bus.on(NOTIFICATION_DELETE, this.notificationDelete);
    },
    beforeUnmount() {
        bus.off(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
        bus.off(NOTIFICATION_ADD, this.notificationAdd);
        bus.off(NOTIFICATION_DELETE, this.notificationDelete);
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
