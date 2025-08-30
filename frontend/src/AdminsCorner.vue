<template>
    <v-container fluid>
      <v-card>
          <v-list>
            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.version') }}</v-list-subheader>
            <v-list-item :title="$vuetify.locale.t('$vuetify.version')" @click="openVersionModal"></v-list-item>

            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.logs') }}</v-list-subheader>
            <v-list-item title="Opensearch Dashboards" href="/opensearch-dashboards" target="_blank"></v-list-item>

            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.tracing') }}</v-list-subheader>
            <v-list-item title="Jaeger" href="/jaeger" target="_blank"></v-list-item>

            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.object_storage') }}</v-list-subheader>
            <v-list-item title="Minio" href="/minio/console" target="_blank"></v-list-item>

            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.queue_broker') }}</v-list-subheader>
            <v-list-item title="RabbitMQ" href="/rabbitmq/" target="_blank"></v-list-item>

            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.database') }}</v-list-subheader>
            <v-list-item title="PostgreSQL" href="/postgresql" target="_blank"></v-list-item>

          </v-list>
      </v-card>
      <v-dialog v-model="showVersion" width="auto" scrollable>
        <v-progress-linear
            :active="loading"
            :indeterminate="loading"
            absolute
            bottom
            color="primary"
        ></v-progress-linear>
        <v-card>
          <v-table fixed-header>
            <thead>
              <tr>
                <th class="text-left">
                  Name
                </th>
                <th class="text-left">
                  Commit Hash
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>aaa</td>
                <td>{{ aaaVersion }}</td>
              </tr>
              <tr>
                <td>chat</td>
                <td>{{ chatVersion }}</td>
              </tr>
              <tr>
                <td>storage</td>
                <td>{{ storageVersion }}</td>
              </tr>
              <tr>
                <td>video</td>
                <td>{{ videoVersion }}</td>
              </tr>
              <tr>
                <td>notification</td>
                <td>{{ notificationVersion }}</td>
              </tr>
              <tr>
                <td>event</td>
                <td>{{ eventVersion }}</td>
              </tr>
              <tr>
                <td>frontend</td>
                <td>{{ frontendVersion }}</td>
              </tr>
              <tr>
                <td>public</td>
                <td>{{ publicVersion }}</td>
              </tr>
            </tbody>
          </v-table>

        </v-card>
      </v-dialog>
    </v-container>
</template>

<script>
    import axios from "axios";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";
    import {setTitle} from "@/utils.js";

    export default {
        data() {
          return {
            showVersion: false,
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
            ...mapStores(useChatStore),
        },
        methods: {
          openVersionModal() {
            this.showVersion = true;
          },
          async loadVersions() {
            this.loading = true;

            const aaaPromise = axios.get(`/aaa/git.json`).then(( {data} ) => {
              this.aaaVersion = data.commit;
            }).catch((e)=>{
              this.aaaVersion = "error: " + e
            })
            const chatPromise = axios.get(`/chat/git.json`).then(( {data} ) => {
              this.chatVersion = data.commit;
            }).catch((e)=>{
              this.chatVersion = "error: " + e
            })
            const storagePromise = axios.get(`/storage/git.json`).then(( {data} ) => {
              this.storageVersion = data.commit;
            }).catch((e)=>{
              this.storageVersion = "error: " + e
            })
            const videoPromise = axios.get(`/video/git.json`).then(( {data} ) => {
              this.videoVersion = data.commit;
            }).catch((e)=>{
              this.videoVersion = "error: " + e
            })
            const notificationPromise = axios.get(`/notification/git.json`).then(( {data} ) => {
              this.notificationVersion = data.commit;
            }).catch((e)=>{
              this.notificationVersion = "error: " + e
            })
            const eventPromise = axios.get(`/event/git.json`).then(( {data} ) => {
              this.eventVersion = data.commit;
            }).catch((e)=>{
              this.eventVersion = "error: " + e
            })
            const frontendPromise = axios.get(`/frontend/git.json`).then(( {data} ) => {
              this.frontendVersion = data.commit;
            }).catch((e)=>{
              this.frontendVersion = "error: " + e
            })
            const publicPromise = axios.get(`/public/git.json`).then(( {data} ) => {
              this.publicVersion = data.commit;
            }).catch((e)=>{
              this.publicVersion = "error: " + e
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
        },
        watch: {
          'showVersion': {
            handler: function (newValue, oldValue) {
              if (newValue) {
                this.loadVersions()
              }
            }
          },
        },
        mounted() {
            const title = this.$vuetify.locale.t('$vuetify.admins_corner');
            this.chatStore.title = title;
            setTitle(title);
        },
        beforeUnmount() {
            this.chatStore.title = null;
        }
    }
</script>

<style scoped lang="stylus">
.ellipsis-wrapper {
  display: block
}
.ellipsis-text {
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}
</style>
