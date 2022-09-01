<template>
    <v-row justify="center">
        <v-dialog v-model="show" persistent scrollable max-width="1280">
            <v-card>
                <v-card-title>Editing '{{filename}}'</v-card-title>

                <v-card-text>
                    <v-progress-circular
                        indeterminate
                        color="primary"
                        v-if="loading"
                    ></v-progress-circular>
                    <v-textarea v-model="editableText" v-else auto-grow autofocus filled dense/>
                </v-card-text>

                <v-card-actions class="pa-4 pt-0">
                    <v-btn color="primary" class="mr-4" @click="saveFile()">Save</v-btn>
                    <v-btn class="mr-4" @click="closeModal()">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {
    OPEN_TEXT_EDIT_MODAL,
    CLOSE_TEXT_EDIT_MODAL,
} from "./bus";
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
                axios.get(fileInfoDto.url).then(response => {
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
        created() {
            bus.$on(OPEN_TEXT_EDIT_MODAL, this.showModal);
            bus.$on(CLOSE_TEXT_EDIT_MODAL, this.closeModal)
        },
        destroyed() {
            bus.$off(OPEN_TEXT_EDIT_MODAL, this.showModal);
            bus.$off(CLOSE_TEXT_EDIT_MODAL, this.closeModal)
        },
    }
</script>