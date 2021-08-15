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
                        :error-messages="limitError"
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
                    <template v-if="!limitError && files.length > 0">
                        <v-btn color="primary" v-if="!uploading" @click="upload()">Upload</v-btn>
                        <v-btn v-else @click="cancel()">Cancel</v-btn>
                    </template>
                    <v-btn class="mr-4" @click="hideModal()" :disabled="uploading">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {OPEN_FILE_UPLOAD_MODAL, CLOSE_FILE_UPLOAD_MODAL, SET_FILE_ITEM_UUID, UPDATE_VIEW_FILES_DIALOG} from "./bus";
import axios from "axios";
import throttle from "lodash/throttle";
const CancelToken = axios.CancelToken;

const formatSize = (size) => {
    const operableSize = Math.abs(size);
    if (operableSize > 1024 * 1024 * 1024 * 1024) {
        return (size / 1024 / 1024 / 1024 / 1024).toFixed(2) + ' TB'
    } else if (operableSize > 1024 * 1024 * 1024) {
        return (size / 1024 / 1024 / 1024).toFixed(2) + ' GB'
    } else if (operableSize > 1024 * 1024) {
        return (size / 1024 / 1024).toFixed(2) + ' MB'
    } else if (operableSize > 1024) {
        return (size / 1024).toFixed(2) + ' KB'
    }
    return size.toString() + ' B'
};

export default {
    data () {
        return {
            uploading: false,
            show: false,
            files: [],
            fileItemUuid: null, // null at first upload, non-nul when user adds files,
            progress: 0,
            cancelSource: null,
            limitError: null,
            shouldSetFileUuidToMessage: false,
            shouldUpdateFileList: false,
        }
    },
    methods: {
        showModal(fileItemUuid, shouldSetFileUuidToMessage, shouldUpdateFileList) {
            this.$data.show = true;
            this.$data.fileItemUuid = fileItemUuid;
            this.$data.shouldSetFileUuidToMessage = shouldSetFileUuidToMessage;
            this.$data.shouldUpdateFileList = shouldUpdateFileList;
        },
        hideModal() {
            this.$data.show = false;
            this.files = [];
            this.progress = 0;
            this.cancelSource = null;
            this.uploading = false;
            this.limitError = null;
            this.$data.fileItemUuid = null;
            this.$data.shouldSetFileUuidToMessage = false;
            this.$data.shouldUpdateFileList = false;
        },
        onProgressFunction(event) {
            this.progress = Math.round((100 * event.loaded) / event.total);
        },
        checkLimits(totalSize) {
            return axios.get(`/api/storage/${this.chatId}/file`, { params: {
                    desiredSize: totalSize,
                }}).then(value => {
                if (value.data.status != "ok") {
                    this.limitError = [`Too large, ${formatSize(value.data.available)} available`];
                    return Promise.reject(this.limitError);
                } else {
                    this.limitError = null;
                    return Promise.resolve();
                }
            });
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
            let totalSize = 0;
            for (const file of this.files) {
                totalSize += file.size;
                formData.append('files', file);
            }
            return this.checkLimits(totalSize).then(()=>{
                return axios.post(`/api/storage/${this.chatId}/file`+(this.fileItemUuid ? `/${this.fileItemUuid}` : ''), formData, config)
                    .then(response => {
                        if (this.$data.shouldSetFileUuidToMessage) {
                            bus.$emit(SET_FILE_ITEM_UUID, {fileItemUuid: response.data.fileItemUuid, count: response.data.count});
                        }
                        this.uploading = false;
                        if (this.$data.shouldUpdateFileList) {
                            bus.$emit(UPDATE_VIEW_FILES_DIALOG);
                        }
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
            }).catch(ex =>{
                this.uploading = false;
                return Promise.reject(ex);
            });
        },
        cancel() {
            this.cancelSource.cancel()
        },
        updateFiles(files) {
            console.log("updateFiles", files);
            this.files = [...files];
            this.limitError = null;
            let totalSize = 0;
            for (const file of this.files) {
                totalSize += file.size;
            }
            if (totalSize > 0) {
                this.checkLimits(totalSize);
            }
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