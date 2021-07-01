<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" persistent>
            <v-card>
                <v-card-title>Upload files</v-card-title>

                <v-container>
                    <v-file-input
                        :disabled="uploading"
                        :value="files"
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
                        v-model="progress"
                        color="success"
                        buffer-value="0"
                        stream
                    >
                    </v-progress-linear>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" v-if="!uploading" @click="upload()">Upload</v-btn>
                    <v-btn v-else @click="cancel()">Cancel</v-btn>
                    <v-btn class="mr-4" @click="hideModal()" :disabled="uploading">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {OPEN_FILE_UPLOAD_MODAL, CLOSE_FILE_UPLOAD_MODAL, SET_FILE_ITEM_UUID} from "./bus";
import axios from "axios";
import throttle from "lodash/throttle";
const CancelToken = axios.CancelToken;

export default {
    data () {
        return {
            uploading: false,
            show: false,
            files: [],
            fileItemUuid: null, // null at first upload, non-nul when user adds files,
            progress: 0,
            cancelSource: null,
        }
    },
    methods: {
        showModal(fileItemUuid) {
            this.$data.show = true;
            this.$data.fileItemUuid = fileItemUuid;
        },
        hideModal() {
            this.$data.show = false;
            this.files = [];
            this.progress = 0;
            this.cancelSource = null;
            this.uploading = false;
        },
        onProgressFunction(event) {
            this.progress = Math.round((100 * event.loaded) / event.total);
        },
        upload() {
            this.uploading = true;
            this.cancelSource = CancelToken.source();
            const config = {
                headers: { 'content-type': 'multipart/form-data' },
                onUploadProgress: this.onProgressFunction,
                cancelToken: this.cancelSource.token
            }
            console.log("Sending file to storage");
            const formData = new FormData();
            for (const file of this.files) {
                formData.append('files', file);
            }
            return axios.post(`/api/storage/${this.chatId}/file`+(this.fileItemUuid ? `/${this.fileItemUuid}` : ''), formData, config)
                .then(response => {
                    bus.$emit(SET_FILE_ITEM_UUID, {fileItemUuid: response.data.fileItemUuid, count: response.data.count});
                    this.uploading = false;
                })
                .catch((thrown) => {
                    if (axios.isCancel(thrown)) {
                        console.log('Request canceled', thrown.message);
                        this.hideModal();
                    } else {
                        throw thrown
                    }
                })
                .then(()=>{this.hideModal();})
        },
        cancel() {
            this.cancelSource.cancel()
        },
        updateFiles(files) {
            console.log("updateFiles", files);
            this.files = [...files];
        }
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
    },
    created() {
        bus.$on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        this.onProgressFunction = throttle(this.onProgressFunction, 100);
    },
    destroyed() {
        bus.$off(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$off(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
    },
}
</script>