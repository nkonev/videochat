<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" persistent>
            <v-card>
                <v-card-title>Upload files</v-card-title>

                <v-container>
                    <v-file-input
                        :disabled="uploading"
                        counter
                        multiple
                        show-size
                        small-chips
                        truncate-length="15"
                        @change="updateFiles"
                    ></v-file-input>

                    <v-progress-linear
                        class="mt-2"
                        v-if="uploading"
                        v-model="totalProgress"
                        color="success"
                        buffer-value="0"
                        value="20"
                        stream
                    >
                    </v-progress-linear>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" :loading="uploading" @click="upload()" :disabled="uploading">Upload</v-btn>
                    <v-btn class="mr-4" @click="hideModal()" :disabled="uploading">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {OPEN_FILE_UPLOAD_MODAL, CLOSE_FILE_UPLOAD_MODAL} from "./bus";
import axios from "axios";

export default {
    data () {
        return {
            uploading: false,
            show: false,
            files: [],
        }
    },
    methods: {
        showModal(chatId) {
            this.$data.show = true;
        },
        hideModal() {
            this.$data.show = false;
            this.files = [];
        },
        upload() {
            this.uploading = true;
            const config = {
                headers: { 'content-type': 'multipart/form-data' }
            }
            console.log("Sending file to storage");
            const formData = new FormData();
            for (const file of this.files) {
                formData.append('files', file);
            }
            return axios.post(`/api/storage/${this.chatId}/file`, formData, config)
                .then(value => {
                    this.uploading = false;
                })
        },
        updateFiles(files) {
            console.log("updateFiles", files);
            this.files = [...this.files, ...files];
        }
    },
    computed: {
        totalProgress() {
            return 25;
        },
        chatId() {
            return this.$route.params.id
        },
    },
    created() {
        bus.$on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
    },
    destroyed() {
        bus.$off(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$off(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
    },
}
</script>