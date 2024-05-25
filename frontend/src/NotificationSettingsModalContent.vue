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
          :title="$vuetify.locale.t('$vuetify.notifications_settings_global')"
          :disabled="loading"
      >
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
              :label="$vuetify.locale.t('$vuetify.notify_about_missed_calls')"
              density="compact"
              color="primary"
              hide-details
              class="ml-4 mr-4 py-1"
              v-model="notificationsSettings.missedCallsEnabled"
              @update:modelValue="putGlobalNotificationsSettings()"
          ></v-switch>
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
              :label="$vuetify.locale.t('$vuetify.notify_about_reactions')"
              density="compact"
              color="primary"
              hide-details
              class="ml-4 mr-4 py-1"
              v-model="notificationsSettings.reactionsEnabled"
              @update:modelValue="putGlobalNotificationsSettings()"
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

      </v-card>
  </v-card-text>

</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import axios from "axios";
    import {chat_name, videochat_name} from "@/router/routes.js";

    export default {
        data () {
            return {
                loading: false,
                notificationsSettings: {},
                notificationsChatSettings: {}
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
            isInChat() {
                return this.$route.name == chat_name || this.$route.name == videochat_name
            },
        },
        mounted() {
            console.debug("Initially set loading true")
            this.loading = true;
            axios.get(`/api/notification/settings/global`).then(( {data} ) => {
                this.notificationsSettings = data;
                console.debug("Loaded notificationsGlobalSetting", this.notificationsSettings)
            }).then(()=>{
                if (this.isInChat()) {
                    return axios.get(`/api/notification/settings/${this.chatId}/chat`).then(({data}) => {
                        this.notificationsChatSettings = data;
                        console.debug("Loaded notificationsChatSetting", this.notificationsChatSettings)
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
