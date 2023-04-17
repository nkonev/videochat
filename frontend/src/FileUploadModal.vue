<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" :persistent="uploading || files.length > 0">
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.upload_files') }}</v-card-title>

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
                        :error-messages="limitError ? [limitError] : []"
                    ></v-file-input>

                    <v-progress-linear
                        class="mt-2"
                        v-if="uploading"
                        v-model="progress"
                        color="success"
                        buffer-value="0"
                        stream
                        light
                        height="25"
                    >
                      <strong>{{ progressLoaded | formatSizeFilter }} / {{ progressTotal | formatSizeFilter }}</strong>
                    </v-progress-linear>
                </v-container>

                <v-card-actions class="pa-4">
                    <template v-if="!limitError && files.length > 0">
                        <v-btn color="primary" v-if="!uploading" @click="upload()">{{ $vuetify.lang.t('$vuetify.upload') }}</v-btn>
                        <v-btn v-else @click="cancel()">{{ $vuetify.lang.t('$vuetify.cancel') }}</v-btn>
                    </template>
                    <v-btn class="mr-4" @click="hideModal()" :disabled="uploading">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {
    OPEN_FILE_UPLOAD_MODAL,
    CLOSE_FILE_UPLOAD_MODAL,
    SET_FILE_ITEM_UUID,
    UPDATE_VIEW_FILES_DIALOG,
    FILE_UPLOAD_MODAL_START_UPLOADING
} from "./bus";
import axios from "axios";
import throttle from "lodash/throttle";
import { formatSize } from "./utils";
const CancelToken = axios.CancelToken;

export default {
    data () {
        return {
            uploading: false,
            show: false,
            files: [],
            fileItemUuid: null, // null at first upload, non-nul when user adds files,
            progress: 0,
            progressTotal: 0,
            progressLoaded: 0,
            cancelSource: null,
            limitError: null,
            shouldSetFileUuidToMessage: false,
            filesWerePredefined: false,
            correlationId: null,
        }
    },
    filters: {
        formatSizeFilter(size) {
            return formatSize((size))
        },
    },
    methods: {
        showModal(fileItemUuid, shouldSetFileUuidToMessage, predefinedFiles, correlationId) {
            this.$data.show = true;
            this.$data.fileItemUuid = fileItemUuid;
            this.$data.shouldSetFileUuidToMessage = shouldSetFileUuidToMessage;
            if (predefinedFiles) {
                this.$data.files = predefinedFiles;
                this.$data.filesWerePredefined = true;
            }
            this.correlationId = correlationId;
            console.log("Opened FileUploadModal with fileItemUuid=", fileItemUuid, ", shouldSetFileUuidToMessage=", shouldSetFileUuidToMessage, ", predefinedFiles=", predefinedFiles, ", correlationId=", correlationId);
        },
        hideModal() {
            this.$data.show = false;
            this.files = [];
            this.progress = 0;
            this.progressTotal = 0;
            this.progressLoaded = 0;
            this.cancelSource = null;
            this.uploading = false;
            this.limitError = null;
            this.$data.fileItemUuid = null;
            this.$data.shouldSetFileUuidToMessage = false;
            this.$data.filesWerePredefined = false;
            this.correlationId = null;
        },
        onProgressFunction(event) {
            this.progress = Math.round((100 * event.loaded) / event.total);
            this.progressLoaded = (event.loaded);
            this.progressTotal = (event.total);
        },
        checkLimits(totalSize) {
            return axios.get(`/api/storage/${this.chatId}/file`, { params: {
                    desiredSize: totalSize,
                }}).then(value => {
                if (value.data.status != "ok") {
                    this.limitError = `Too large, ${formatSize(value.data.available)} available`;
                    return Promise.reject(this.limitError);
                } else {
                    this.limitError = null;
                    return Promise.resolve();
                }
            });
        },
        async upload() {
            this.uploading = true;
            this.cancelSource = CancelToken.source();
            const config = {
                // headers: { 'content-type': 'multipart/form-data' },
                onUploadProgress: this.onProgressFunction,
                cancelToken: this.cancelSource.token
            }
            console.log("Sending file to storage");

            let totalSize = 0;
            const urlResponses = [];
            for (const file of this.files) {
                totalSize += file.size;
                const response = await axios.put(`/api/storage/${this.chatId}/url`, {
                    fileItemUuid: this.fileItemUuid, // nullable
                    fileSize: file.size,
                    fileName: file.name,
                    correlationId: this.correlationId, // nullable
                })
                this.fileItemUuid = response.data.fileItemUuid;
                urlResponses.push({
                    url: response.data.url,
                    file: file,
                    existingCount: response.data.existingCount,
                });
            }

            try {
                await this.checkLimits(totalSize)
            } catch (errMsg) {
                this.uploading = false;
                return Promise.resolve();
            }

            for (const [index, presignedUrlResponse] of urlResponses.entries()) {
                try {
                    await axios.put(presignedUrlResponse.url, presignedUrlResponse.file, config)
                        .then(response => {
                            if (this.$data.shouldSetFileUuidToMessage) {
                                bus.$emit(SET_FILE_ITEM_UUID, {
                                    fileItemUuid: this.fileItemUuid,
                                    count: (presignedUrlResponse.existingCount + index + 1)
                                });
                            }
                            bus.$emit(UPDATE_VIEW_FILES_DIALOG);
                            return response;
                        })
                } catch(thrown) {
                    this.uploading = false;
                    if (axios.isCancel(thrown)) {
                        console.log('Request canceled', thrown.message);
                        break
                    } else {
                        return Promise.reject(thrown);
                    }
                }
            }
            this.uploading = false;
            this.hideModal();
            return Promise.resolve();
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
        bus.$on(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
        this.onProgressFunction = throttle(this.onProgressFunction, 100);
    },
    destroyed() {
        bus.$off(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.$off(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        bus.$off(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
    },
    watch: {
        show(newValue) {
            if (!newValue) {
                this.hideModal();
            }
        }
    }
}
</script>
