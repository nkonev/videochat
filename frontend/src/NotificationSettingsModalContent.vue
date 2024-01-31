<template>

  <v-card-text class="pb-0">

      <v-progress-linear
          :active="loading"
          :indeterminate="loading"
          absolute
          bottom
          color="primary"
      ></v-progress-linear>

      <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_mentions')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="chatStore.notificationsSettings.mentionsEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
      <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_missed_calls')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="chatStore.notificationsSettings.missedCallsEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
      <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_replies')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="chatStore.notificationsSettings.answersEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
    <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_reactions')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="chatStore.notificationsSettings.reactionsEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
    ></v-switch>

  </v-card-text>

</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";
    import axios from "axios";

    export default {
        data () {
            return {
                language: null,
                loading: false,
            }
        },
        computed: {
            ...mapStores(useChatStore),
        },
        methods: {
            putNotificationsSettings() {
                this.loading = true;
                axios.put('/api/notification/settings', this.chatStore.notificationsSettings).then(({data}) => {
                    this.chatStore.notificationsSettings = data;
                    console.log("Stored notificationsSetting", data)
                }).finally(()=>{
                    this.loading = false;
                })
            },
        },
        mounted() {
            console.log("Loaded notificationsSetting", this.chatStore.notificationsSettings)
        }
    }
</script>
