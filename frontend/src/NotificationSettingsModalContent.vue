<template>

  <v-card-text class="pb-0 notification-settings-wrapper">

      <v-progress-linear
          :active="loading"
          :indeterminate="loading"
          absolute
          bottom
          color="primary"
      ></v-progress-linear>

      <v-card
          rounded
          border
          class="notification-global mb-1"
          :disabled="loading"
      >
          <v-card-title class="d-flex align-center">
              <span>{{ $vuetify.locale.t('$vuetify.notifications_settings_global') }}</span>
              <v-spacer/>
              <v-btn v-if="!isMobile()" variant="outlined" @click="onEnableNotification">{{ $vuetify.locale.t('$vuetify.request_notifications_in_browser') }}</v-btn>
          </v-card-title>

          <span class="d-flex align-center">
              <v-switch
                  :label="$vuetify.locale.t('$vuetify.notify_about_mentions')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 pb-1"
                  v-model="notificationsSettings.mentionsEnabled"
                  @update:modelValue="putGlobalNotificationsSettings()"
              ></v-switch>
              <v-switch
                  v-if="!isMobile()"
                  :label="$vuetify.locale.t('$vuetify.notifications_in_browser')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 pb-1"
                  v-model="browserNotificationSettings.mentionsEnabled"
                  @update:modelValue="putGlobalNotificationBrowserSettings()"
              ></v-switch>
          </span>

          <span class="d-flex align-center">
              <v-switch
                  :label="$vuetify.locale.t('$vuetify.notify_about_missed_calls')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 py-1"
                  v-model="notificationsSettings.missedCallsEnabled"
                  @update:modelValue="putGlobalNotificationsSettings()"
              ></v-switch>
              <v-switch
                  v-if="!isMobile()"
                  :label="$vuetify.locale.t('$vuetify.notifications_in_browser')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 pb-1"
                  v-model="browserNotificationSettings.missedCallsEnabled"
                  @update:modelValue="putGlobalNotificationBrowserSettings()"
              ></v-switch>
          </span>

          <span class="d-flex align-center">
              <v-switch
                  :label="$vuetify.locale.t('$vuetify.notify_about_replies')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 py-1"
                  v-model="notificationsSettings.answersEnabled"
                  @update:modelValue="putGlobalNotificationsSettings()"
              ></v-switch>
              <v-switch
                  v-if="!isMobile()"
                  :label="$vuetify.locale.t('$vuetify.notifications_in_browser')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 pb-1"
                  v-model="browserNotificationSettings.answersEnabled"
                  @update:modelValue="putGlobalNotificationBrowserSettings()"
              ></v-switch>
          </span>

          <span class="d-flex align-center">
              <v-switch
                  :label="$vuetify.locale.t('$vuetify.notify_about_reactions')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 py-1"
                  v-model="notificationsSettings.reactionsEnabled"
                  @update:modelValue="putGlobalNotificationsSettings()"
              ></v-switch>
              <v-switch
                  v-if="!isMobile()"
                  :label="$vuetify.locale.t('$vuetify.notifications_in_browser')"
                  density="compact"
                  color="primary"
                  hide-details
                  class="ml-4 mr-4 pb-1"
                  v-model="browserNotificationSettings.reactionsEnabled"
                  @update:modelValue="putGlobalNotificationBrowserSettings()"
              ></v-switch>
          </span>
          <v-switch
              v-if="!isMobile()"
              :label="$vuetify.locale.t('$vuetify.new_message_notifications_in_browser')"
              density="compact"
              color="primary"
              hide-details
              class="ml-4 mr-4 pb-1"
              v-model="browserNotificationSettings.newMessagesEnabled"
              @update:modelValue="putGlobalNotificationBrowserSettings()"
          ></v-switch>
          <v-switch
              v-if="!isMobile()"
              :label="$vuetify.locale.t('$vuetify.call_notifications_in_browser')"
              density="compact"
              color="primary"
              hide-details
              class="ml-4 mr-4 pb-1"
              v-model="browserNotificationSettings.callEnabled"
              @update:modelValue="putGlobalNotificationBrowserSettings()"
          ></v-switch>

      </v-card>

      <v-card
          :disabled="!isInChat() || loading"
          rounded
          border
          class="notification-overrides mb-1"
          :title="$vuetify.locale.t('$vuetify.notifications_settings_per_chat_override')"
      >
          <v-radio-group inline
                         :label="$vuetify.locale.t('$vuetify.notify_about_mentions')"
                         color="primary"
                         hide-details
                         class="mb-2"
                         v-model="notificationsChatSettings.mentionsEnabled"
                         @update:modelValue="putPerChatNotificationsSettings()"
          >
              <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
          </v-radio-group>

          <v-radio-group inline
                         :label="$vuetify.locale.t('$vuetify.notify_about_missed_calls')"
                         color="primary"
                         hide-details
                         class="mb-2"
                         v-model="notificationsChatSettings.missedCallsEnabled"
                         @update:modelValue="putPerChatNotificationsSettings()"
          >
              <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
          </v-radio-group>

          <v-radio-group inline
                         :label="$vuetify.locale.t('$vuetify.notify_about_replies')"
                         color="primary"
                         hide-details
                         class="mb-2"
                         v-model="notificationsChatSettings.answersEnabled"
                         @update:modelValue="putPerChatNotificationsSettings()"
          >
              <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
          </v-radio-group>

          <v-radio-group inline
                         :label="$vuetify.locale.t('$vuetify.notify_about_reactions')"
                         color="primary"
                         hide-details
                         class="mb-2"
                         v-model="notificationsChatSettings.reactionsEnabled"
                         @update:modelValue="putPerChatNotificationsSettings()"
          >
              <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
          </v-radio-group>

          <v-radio-group inline
                         :label="$vuetify.locale.t('$vuetify.consider_messages_of_this_chat_as_unread')"
                         color="primary"
                         hide-details
                         class="mb-2"
                         v-model="considerMessagesOfThisChatAsUnread"
                         @update:modelValue="putConsiderMessagesOfThisChatAsUnread()"
          >
              <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
              <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
          </v-radio-group>

          <template v-if="!isMobile()">
              <v-radio-group inline
                             :label="$vuetify.locale.t('$vuetify.new_message_notifications_in_browser')"
                             color="primary"
                             hide-details
                             class="mb-2"
                             v-model="browserNotificationChatSettings.newMessagesEnabled"
                             @update:modelValue="putChatNotificationBrowserSettings()"
              >
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
              </v-radio-group>

              <v-radio-group inline
                             :label="$vuetify.locale.t('$vuetify.call_notifications_in_browser')"
                             color="primary"
                             hide-details
                             class="mb-2"
                             v-model="browserNotificationChatSettings.callEnabled"
                             @update:modelValue="putChatNotificationBrowserSettings()"
              >
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_not_set')" :value="null"></v-radio>
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_on')" :value="true"></v-radio>
                  <v-radio :label="$vuetify.locale.t('$vuetify.option_off')" :value="false"></v-radio>
              </v-radio-group>
          </template>
      </v-card>
  </v-card-text>

</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import axios from "axios";
    import {chat_name, videochat_name} from "@/router/routes.js";
    import {createBrowserNotification} from "@/browserNotifications.js";
    import {
        getBrowserNotification,
        getGlobalBrowserNotification,
        NOTIFICATION_TYPE_ANSWERS, NOTIFICATION_TYPE_CALL,
        NOTIFICATION_TYPE_MENTIONS,
        NOTIFICATION_TYPE_MISSED_CALLS, NOTIFICATION_TYPE_NEW_MESSAGES,
        NOTIFICATION_TYPE_REACTIONS,
        setBrowserNotification,
        setGlobalBrowserNotification
    } from "@/store/localStore.js";

    export default {
        data () {
            return {
                loading: false,
                notificationsSettings: {},
                notificationsChatSettings: {},
                considerMessagesOfThisChatAsUnread: null,
                browserNotificationSettings: {},
                browserNotificationChatSettings: {},
            }
        },
        computed: {
            ...mapStores(useChatStore),
            chatId() {
                return this.$route.params.id
            },
        },
        methods: {
            putGlobalNotificationsSettings() {
                this.loading = true;
                axios.put('/api/notification/settings/global', this.notificationsSettings).then(({data}) => {
                    this.notificationsSettings = data;
                }).finally(()=>{
                    this.loading = false;
                })
            },
            putPerChatNotificationsSettings() {
                this.loading = true;
                axios.put(`/api/notification/settings/${this.chatId}/chat`, this.notificationsChatSettings).then(({data}) => {
                    this.notificationsChatSettings = data;
                }).finally(()=>{
                    this.loading = false;
                })
            },
            putConsiderMessagesOfThisChatAsUnread() {
                this.loading = true;
                axios.put(`/api/chat/${this.chatId}/notification`, {considerMessagesOfThisChatAsUnread: this.considerMessagesOfThisChatAsUnread}).then(({data}) => {
                    this.considerMessagesOfThisChatAsUnread = data.considerMessagesOfThisChatAsUnread;
                }).finally(()=>{
                    this.loading = false;
                })
            },
            isInChat() {
                return this.$route.name == chat_name || this.$route.name == videochat_name
            },
            onEnableNotification() {
                Notification.requestPermission().then((result) => {
                    createBrowserNotification(this.$vuetify.locale.t('$vuetify.notifications_title'), this.$vuetify.locale.t('$vuetify.notifications_were_permitted'), "message")
                })
            },
            setGlobalNotificationsToFalse() {
                this.browserNotificationSettings.mentionsEnabled = false;
                this.browserNotificationSettings.missedCallsEnabled = false;
                this.browserNotificationSettings.answersEnabled = false;
                this.browserNotificationSettings.reactionsEnabled = false;
                this.browserNotificationSettings.newMessagesEnabled = false;
                this.browserNotificationSettings.callEnabled = false;
            },
            writeGlobalNotificationsToLocalStore() {
                setGlobalBrowserNotification(NOTIFICATION_TYPE_MENTIONS, this.browserNotificationSettings.mentionsEnabled);
                setGlobalBrowserNotification(NOTIFICATION_TYPE_MISSED_CALLS, this.browserNotificationSettings.missedCallsEnabled);
                setGlobalBrowserNotification(NOTIFICATION_TYPE_ANSWERS, this.browserNotificationSettings.answersEnabled);
                setGlobalBrowserNotification(NOTIFICATION_TYPE_REACTIONS, this.browserNotificationSettings.reactionsEnabled);
                setGlobalBrowserNotification(NOTIFICATION_TYPE_NEW_MESSAGES, this.browserNotificationSettings.newMessagesEnabled);
                setGlobalBrowserNotification(NOTIFICATION_TYPE_CALL, this.browserNotificationSettings.callEnabled);
            },
            readGlobalNotificationsFromLocalStore() {
                this.browserNotificationSettings.mentionsEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_MENTIONS)
                this.browserNotificationSettings.missedCallsEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_MISSED_CALLS)
                this.browserNotificationSettings.answersEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_ANSWERS)
                this.browserNotificationSettings.reactionsEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_REACTIONS)
                this.browserNotificationSettings.newMessagesEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_NEW_MESSAGES)
                this.browserNotificationSettings.callEnabled = getGlobalBrowserNotification(NOTIFICATION_TYPE_CALL)
            },

            setChatNotificationsToFalse() {
                this.browserNotificationChatSettings.newMessagesEnabled = false;
                this.browserNotificationChatSettings.callEnabled = false;
            },
            writeChatNotificationsToLocalStore() {
                setBrowserNotification(this.chatId, NOTIFICATION_TYPE_NEW_MESSAGES, this.browserNotificationChatSettings.newMessagesEnabled);
                setBrowserNotification(this.chatId, NOTIFICATION_TYPE_CALL, this.browserNotificationChatSettings.callEnabled);
            },
            readChatNotificationsFromLocalStore() {
                this.browserNotificationChatSettings.newMessagesEnabled = getBrowserNotification(this.chatId, null, NOTIFICATION_TYPE_NEW_MESSAGES)
                this.browserNotificationChatSettings.callEnabled = getBrowserNotification(this.chatId, null, NOTIFICATION_TYPE_CALL)
            },

            putGlobalNotificationBrowserSettings() {
                Notification.requestPermission().then((permission) => {
                    if (permission !== "granted") {
                        this.setGlobalNotificationsToFalse();
                    }
                    this.writeGlobalNotificationsToLocalStore();
                })
            },
            putChatNotificationBrowserSettings() {
                Notification.requestPermission().then((permission) => {
                    if (permission !== "granted") {
                        this.setChatNotificationsToFalse();
                    }
                    this.writeChatNotificationsToLocalStore();
                })
            },
        },
        mounted() {
            console.debug("Initially set loading true")
            this.loading = true;
            axios.get(`/api/notification/settings/global`).then(( {data} ) => {
                this.notificationsSettings = data;
                console.debug("Loaded notificationsGlobalSetting", this.notificationsSettings)
            }).then(()=>{
                if (Notification?.permission !== "granted") {
                    this.setGlobalNotificationsToFalse();
                    this.writeGlobalNotificationsToLocalStore();
                } else {
                    this.readGlobalNotificationsFromLocalStore();
                }

                if (this.isInChat()) {
                    return axios.get(`/api/notification/settings/${this.chatId}/chat`).then(({data}) => {
                        this.notificationsChatSettings = data;
                        console.debug("Loaded notificationsChatSetting", this.notificationsChatSettings)
                    });
                } else {
                    return Promise.resolve()
                }
            }).then(()=>{
                if (this.isInChat()) {
                    return axios.get(`/api/chat/${this.chatId}/notification`).then(({data}) => {
                        this.considerMessagesOfThisChatAsUnread = data.considerMessagesOfThisChatAsUnread;
                        console.debug("Loaded considerMessagesOfThisChatAsUnread", this.considerMessagesOfThisChatAsUnread);

                        if (Notification?.permission !== "granted") {
                            this.setChatNotificationsToFalse();
                            this.writeChatNotificationsToLocalStore();
                        } else {
                            this.readChatNotificationsFromLocalStore();
                        }

                    });
                } else {
                    return Promise.resolve()
                }
            }).then(()=>{
                console.debug("Finally set loading false")
                this.loading = false;
            })
        }
    }
</script>

<style lang="stylus">
.notification-settings-wrapper {
    .v-card-item {
        padding-bottom 0
    }

    .notification-overrides {
        .v-radio-group > .v-input__control {
            margin-top 0.4rem
        }

        .v-selection-control--density-default {
            --v-selection-control-size: 32px;
        }

        .v-radio-group > .v-input__control > .v-label + .v-selection-control-group {
            margin-top: 0
            padding-inline-start: 0;
            margin-inline-start: 0.7rem
        }

        .v-selection-control--density-default {
            margin-right 0.4rem
        }
    }

    .notification-global {
        .v-switch .v-label {
            padding-inline-start: 0;
            margin-inline-start: 0.9rem;
        }
    }
}

</style>
