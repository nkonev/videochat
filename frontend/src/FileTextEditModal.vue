<template>
    <v-row justify="center">
        <v-dialog v-model="show" persistent scrollable max-width="800">
            <v-card>
                <v-card-title>{{ this.$vuetify.locale.t('$vuetify.file_editing', filename) }}</v-card-title>

                <v-card-text>
                    <v-progress-circular
                        indeterminate
                        color="primary"
                        v-if="loading"
                    ></v-progress-circular>
                    <v-textarea v-else v-model="editableText" auto-grow autofocus filled dense hide-spin-buttons hide-details/>
                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn variant="flat" color="primary" @click="saveFile()">{{$vuetify.locale.t('$vuetify.ok')}}</v-btn>
                    <v-btn variant="flat" color="red" @click="closeModal()">{{$vuetify.locale.t('$vuetify.close')}}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {
        OPEN_TEXT_EDIT_MODAL,
    } from "./bus/bus";
    import axios from "axios";

    export default {
        data() {
            return {
                show: false,
                editableText: null,
                filename: null,
                loading: false,
                fileId: null,
                chatId: null,
                fileItemUuid: null,
            }
        },
        methods: {
            showModal({fileInfoDto, chatId, fileItemUuid}) {
                this.show = true;
                this.chatId = chatId;
                this.fileItemUuid = fileItemUuid;
                this.filename = fileInfoDto.filename;
                this.fileId = fileInfoDto.id;
                this.loading = true;
                const url = new URL( location.origin + fileInfoDto.url);
                url.searchParams.append('cache', false);
                axios.get(url).then(response => {
                    this.editableText = response.data;
                }).finally(() => {
                    this.loading = false;
                })
            },
            closeModal() {
                this.show = false;
                this.chatId = null;
                this.editableText = null;
                this.loading = false;
                this.filename = null;
                this.fileId = null;
                this.fileItemUuid = null;
            },
            saveFile() {
                this.loading = true;
                return axios.put(`/api/storage/${this.chatId}/replace/file`, {
                    id: this.fileId,
                    text: this.editableText,
                    contentType: "text/plain",
                    filename: this.filename,
                })
                    .then(response => {
                        this.loading = false;
                    })
            },
        },
        mounted() {
            bus.on(OPEN_TEXT_EDIT_MODAL, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_TEXT_EDIT_MODAL, this.showModal);
        },
    }
</script>
