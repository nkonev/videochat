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
          v-model="notificationsSettings.mentionsEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
      <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_missed_calls')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="notificationsSettings.missedCallsEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
      <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_replies')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="notificationsSettings.answersEnabled"
          @update:modelValue="putNotificationsSettings()"
          :disabled="loading"
      ></v-switch>
    <v-switch
          :label="$vuetify.locale.t('$vuetify.notify_about_reactions')"
          density="comfortable"
          color="primary"
          hide-details
          class="ma-0 ml-2 mr-4 py-1"
          v-model="notificationsSettings.reactionsEnabled"
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
                loading: false,
                notificationsSettings: {}
            }
        },
        computed: {
            ...mapStores(useChatStore),
        },
        methods: {
            putNotificationsSettings() {
                this.loading = true;
                axios.put('/api/notification/settings', this.notificationsSettings).then(({data}) => {
                    this.notificationsSettings = data;
                    console.log("Stored notificationsSetting", data)
                }).finally(()=>{
                    this.loading = false;
                })
            },
        },
        mounted() {
            axios.get(`/api/notification/settings`).then(( {data} ) => {
                console.debug("fetched notifications settings =", data);
                this.notificationsSettings = data;
            });
            console.log("Loaded notificationsSetting", this.notificationsSettings)
        }
    }
</script>
