<template>
    <v-container fluid>
    <v-card>
        <v-list>
            <v-list-subheader>{{ $vuetify.locale.t('$vuetify.tracing') }}</v-list-subheader>
            <v-list-item title="Jaeger" href="http://google.com"></v-list-item>
        </v-list>
    </v-card>
    </v-container>
</template>

<script>
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore.js";

    export default {
        computed: {
            ...mapStores(useChatStore),
        },
        mounted() {
            this.chatStore.title = this.$vuetify.locale.t('$vuetify.admins_corner');
        },
        beforeUnmount() {
            this.chatStore.title = null;
        }
    }
</script>
