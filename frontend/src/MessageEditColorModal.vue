<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640">
            <v-card>
                <v-card-title>{{ getTitle() }}</v-card-title>

                <v-card-text class="px-0 py-0">
                    <v-color-picker
                        dot-size="25"
                        hide-canvas
                        hide-inputs
                        hide-sliders
                        show-swatches
                        swatches-max-height="300"
                        v-model="color"
                        width="100%"
                    ></v-color-picker>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-spacer/>
                    <v-btn color="primary" class="mr-2" @click="accept()">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    <v-btn class="mr-2" @click="clear()">{{ $vuetify.lang.t('$vuetify.clear') }}</v-btn>
                    <v-btn color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {MESSAGE_EDIT_COLOR_SET, OPEN_MESSAGE_EDIT_COLOR} from "./bus";
    import {colorBackground, colorText} from "@/utils";

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
            showModal(colorMode, color) {
                this.$data.show = true;
                this.colorMode = colorMode;
                this.color = color;
            },
            accept() {
                bus.$emit(MESSAGE_EDIT_COLOR_SET, this.color, this.colorMode);
                this.closeModal();
            },
            clear() {
                bus.$emit(MESSAGE_EDIT_COLOR_SET, null, this.colorMode);
                this.closeModal();
            },
            closeModal() {
                this.show = false;
                this.colorMode = null;
                this.color = null;
            },
            getTitle() {
                if (this.colorMode == colorText) {
                    return this.$vuetify.lang.t('$vuetify.message_edit_text_color');
                } else if (this.colorMode == colorBackground) {
                    return this.$vuetify.lang.t('$vuetify.message_edit_background_color');
                }
            }
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_COLOR, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_COLOR, this.showModal);
        },
    }
</script>