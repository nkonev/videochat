<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card :title="getTitle()">
                <v-color-picker
                    dot-size="25"
                    hide-canvas
                    hide-inputs
                    hide-sliders
                    show-swatches
                    swatches-max-height="300"
                    :elevation="0"
                    v-model="color"
                    width="100%"
                ></v-color-picker>
                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="primary" variant="flat" @click="accept()">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn @click="clear()" variant="outlined">{{ $vuetify.locale.t('$vuetify.clear') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {COLOR_SET, OPEN_CHOOSE_COLOR} from "./bus/bus";
    import {colorBackground, colorLogin, colorText} from "@/utils";

    export default {
        data () {
            return {
                show: false,
                colorMode: null,
                color: null,
            }
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            }
        },
        methods: {
            showModal({colorMode, color}) {
                this.$data.show = true;
                this.colorMode = colorMode;
                this.color = color;
            },
            accept() {
                bus.emit(COLOR_SET, {color: this.color, colorMode: this.colorMode});
                this.closeModal();
            },
            clear() {
                bus.emit(COLOR_SET, {color: null, colorMode: this.colorMode});
                this.closeModal();
            },
            closeModal() {
                this.show = false;
                this.colorMode = null;
                this.color = null;
            },
            getTitle() {
                if (this.colorMode == colorText) {
                    return this.$vuetify.locale.t('$vuetify.message_edit_text_color');
                } else if (this.colorMode == colorBackground) {
                    return this.$vuetify.locale.t('$vuetify.message_edit_background_color');
                } else if (this.colorMode == colorLogin) {
                    return this.$vuetify.locale.t('$vuetify.login_color');
                }
            }
        },
        mounted() {
            bus.on(OPEN_CHOOSE_COLOR, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_CHOOSE_COLOR, this.showModal);
        },
    }
</script>
