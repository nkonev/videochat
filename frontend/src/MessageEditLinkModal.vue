<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640">
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.message_edit_link') }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-text-field autofocus v-model="link" placeholder="https://google.com"/>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-spacer/>
                    <v-btn color="primary" class="mr-4" @click="accept()">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    <v-btn class="mr-4" @click="clear()">{{ $vuetify.lang.t('$vuetify.clear') }}</v-btn>
                    <v-btn color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {MESSAGE_EDIT_LINK_SET, OPEN_MESSAGE_EDIT_LINK} from "./bus";

    export default {
        data () {
            return {
                show: false,
                link: null,
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
            showModal(url) {
                this.$data.show = true;
                this.link = url;
            },
            accept() {
                bus.$emit(MESSAGE_EDIT_LINK_SET, this.link);
                this.closeModal();
            },
            clear() {
                bus.$emit(MESSAGE_EDIT_LINK_SET, '');
                this.closeModal();
            },
            closeModal() {
                this.show = false;
                this.link = null;
            }
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
    }
</script>