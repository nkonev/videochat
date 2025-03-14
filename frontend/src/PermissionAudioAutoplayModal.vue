<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card :title="$vuetify.locale.t('$vuetify.audio_autoplay_permissions_title')">
                <v-card-text>{{ $vuetify.locale.t('$vuetify.audio_autoplay_permissions_text') }}</v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="primary" variant="flat" @click="onClose">
                        {{ $vuetify.locale.t('$vuetify.ok') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {ON_PERMISSION_AUDIO_AUTOPLAY_GRANTED, OPEN_PERMISSION_AUDIO_AUTOPLAY_MODAL} from "./bus/bus";

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
            onClose() {
                this.show=false;
                bus.emit(ON_PERMISSION_AUDIO_AUTOPLAY_GRANTED)
            },
        },
        mounted() {
            bus.on(OPEN_PERMISSION_AUDIO_AUTOPLAY_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_PERMISSION_AUDIO_AUTOPLAY_MODAL, this.showModal);
        },
    }
</script>
