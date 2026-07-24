<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" persistent>
            <v-card>
                <v-card-text>{{ $vuetify.locale.t('$vuetify.outdated_frontend_modal_text') }}</v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="primary" variant="flat" @click="refreshPage()">
                        {{ $vuetify.locale.t('$vuetify.refresh') }}
                    </v-btn>
                    <v-btn color="red" variant="flat" @click="closeModal">
                      {{ $vuetify.locale.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_OUTDATED_FRONTEND_MODAL, CLOSE_OUTDATED_FRONTEND_MODAL} from "./bus/bus";

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
            refreshPage() {
              location.reload();
            },
            closeModal() {
                this.show=false
            },
        },
        mounted() {
            bus.on(OPEN_OUTDATED_FRONTEND_MODAL, this.showModal);
            bus.on(CLOSE_OUTDATED_FRONTEND_MODAL, this.closeModal);
        },
        beforeUnmount() {
            bus.off(OPEN_OUTDATED_FRONTEND_MODAL, this.showModal);
            bus.off(CLOSE_OUTDATED_FRONTEND_MODAL, this.closeModal);
        },
    }
</script>
