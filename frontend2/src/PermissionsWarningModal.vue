<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.audio_autoplay_permissions_title')">
                <v-card-text>{{ $vuetify.locale.t('$vuetify.audio_autoplay_permissions_text') }}</v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn class="mr-4" @click="show=false">
                        {{ $vuetify.locale.t('$vuetify.close') }}
                    </v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_PERMISSIONS_WARNING_MODAL} from "./bus/bus";

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
        mounted() {
            bus.on(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_PERMISSIONS_WARNING_MODAL, this.showModal);
        },
    }
</script>
