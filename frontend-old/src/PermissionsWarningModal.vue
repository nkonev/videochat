<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.audio_autoplay_permissions_title') }}</v-card-title>

                <v-card-text>{{ $vuetify.lang.t('$vuetify.audio_autoplay_permissions_text') }}</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn class="mr-4" @click="show=false">
                        {{ $vuetify.lang.t('$vuetify.close') }}
                    </v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_PERMISSIONS_WARNING_MODAL} from "./bus";

    export default {
        data () {
            return {
                show: false,
            }
        },
        methods: {
            showModal() {
                this.$data.show = true;
            },
        },
        created() {
            bus.$on(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
    }
</script>