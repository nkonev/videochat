<template>
    <v-progress-linear
        :active="loading"
        :indeterminate="loading"
        absolute
        bottom
        color="primary"
    ></v-progress-linear>
    <v-card>
      <v-list>
        <v-list-item>
          <v-list-item-title>aaa</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ aaaVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>chat</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ chatVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>storage</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ storageVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>video</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ videoVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>notification</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ notificationVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>event</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ eventVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>frontend</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ frontendVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

        <v-list-item>
          <v-list-item-title>public</v-list-item-title>
          <template v-slot:append>
            <v-list-item-action class="align-end">
              {{ publicVersion }}
            </v-list-item-action>
          </template>
        </v-list-item>

      </v-list>
    </v-card>
</template>

<script>

import axios from "axios";

export default {
    data() {
        return {
            loading: false,
            aaaVersion: null,
            chatVersion: null,
            storageVersion: null,
            videoVersion: null,
            notificationVersion: null,
            eventVersion: null,
            frontendVersion: null,
            publicVersion: null,
        }
    },
    computed: {
    },
    methods: {

    },
    async mounted() {
      this.loading = true;

      const aaaPromise = axios.get(`/aaa/git.json`).then(( {data} ) => {
        this.aaaVersion = data.commit;
      })
      const chatPromise = axios.get(`/chat/git.json`).then(( {data} ) => {
        this.chatVersion = data.commit;
      })
      const storagePromise = axios.get(`/storage/git.json`).then(( {data} ) => {
        this.storageVersion = data.commit;
      })
      const videoPromise = axios.get(`/video/git.json`).then(( {data} ) => {
        this.videoVersion = data.commit;
      })
      const notificationPromise = axios.get(`/notification/git.json`).then(( {data} ) => {
        this.notificationVersion = data.commit;
      })
      const eventPromise = axios.get(`/event/git.json`).then(( {data} ) => {
        this.eventVersion = data.commit;
      })
      const frontendPromise = axios.get(`/frontend/git.json`).then(( {data} ) => {
        this.frontendVersion = data.commit;
      })
      const publicPromise = axios.get(`/public/git.json`).then(( {data} ) => {
        this.publicVersion = data.commit;
      })

      const allPromises = [
        aaaPromise,
        chatPromise,
        storagePromise,
        videoPromise,
        notificationPromise,
        eventPromise,
        frontendPromise,
        publicPromise,
      ];

      await Promise.all(allPromises);
      this.loading = false;
    },
    beforeUnmount() {
    },
}
</script>
