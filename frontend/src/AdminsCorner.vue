<template>
    <v-container fluid>
    <v-card>
        <v-list>
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
    </v-container>
</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";
    import {setTitle} from "@/utils.js";

    export default {
        computed: {
            ...mapStores(useChatStore),
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
