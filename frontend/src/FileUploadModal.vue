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

export default {
    data () {
        return {
            uploading: false,
            show: false,
        }
    },
    methods: {
        showModal(chatId) {
            this.$data.show = true;
        },
        hideModal() {
            this.$data.show = false;
        },
        upload() {
            this.uploading = true;
            setTimeout(()=>{
                this.uploading = false;
            }, 5000);
        }
    },
    computed: {
        totalProgress() {
            return 25;
        }
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