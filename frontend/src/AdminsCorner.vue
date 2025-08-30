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
                  Service
                </th>
                <th class="text-left">
                  Commit
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                  v-for="item in serviceVersions"
                  :key="item.name"
              >
                <td>{{ item.name }}</td>
                <td>{{ item.version }}</td>
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

            serviceVersions: [
              {name: 'aaa', version: null},
              {name: 'chat', version: null},
              {name: 'frontend', version: null},
              {name: 'storage', version: null},
              {name: 'video', version: null},
              {name: 'event', version: null},
              {name: 'notification', version: null},
              {name: 'public', version: null},
            ],
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

            const tasks = this.serviceVersions.map(sv => {
              return () => {
                  return axios.get(`/${sv.name}/git.json`).then(({data})=>{
                    sv.version = data.commit;
                  }).catch((e)=>{
                    sv.version = "error: " + e
                  })
              }
            })

            await Promise.all(tasks.map(task => new Promise((resolve, reject) => {
              task().then(()=>{
                resolve()
              }).catch(e=>{
                reject();
              });
            })));

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
