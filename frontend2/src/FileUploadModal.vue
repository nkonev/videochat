<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" :persistent="uploading || files.length > 0">
            <v-card :title="$vuetify.locale.t('$vuetify.upload_files')">

                <v-container>
                    <v-file-input
                        :disabled="uploading"
                        :value="files"
                        counter
                        multiple
                        show-size
                        small-chips
                        truncate-length="15"
                        @update:modelValue="updateChosenFiles"
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
                      <strong>{{ formattedProgress }}</strong>
                    </v-progress-linear>
                </v-container>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <template v-if="!limitError && files.length > 0">
                        <v-btn v-if="!uploading" color="primary" class="mr-2 my-1" @click="upload()">{{ $vuetify.locale.t('$vuetify.upload') }}</v-btn>
                        <v-btn v-else class="mr-2 my-1" @click="cancel()">{{ $vuetify.locale.t('$vuetify.cancel') }}</v-btn>
                    </template>
                    <v-btn class="my-1" color="error" @click="hideModal()" :disabled="uploading">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
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
    FILE_UPLOAD_MODAL_START_UPLOADING
} from "./bus/bus";
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
    methods: {
        showModal({fileItemUuid, shouldSetFileUuidToMessage, predefinedFiles, correlationId}) {
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

            let totalSize = 0;
            for (const file of this.files) {
                totalSize += file.size;
            }

            try {
                await this.checkLimits(totalSize)
            } catch (errMsg) {
                this.uploading = false;
                return Promise.resolve();
            }

            this.cancelSource = CancelToken.source();
            const config = {
                // headers: { 'content-type': 'multipart/form-data' },
                onUploadProgress: this.onProgressFunction,
                cancelToken: this.cancelSource.token
            }
            console.log("Sending file to storage");

            const urlResponses = [];
            for (const file of this.files) {
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
                    newFileName: response.data.newFileName,
                    existingCount: response.data.existingCount,
                });
            }

            for (const [index, presignedUrlResponse] of urlResponses.entries()) {
                try {
                    const formData = new FormData();
                    formData.append('File', presignedUrlResponse.file, presignedUrlResponse.newFileName);
                    const renamedFile = formData.get('File');

                    await axios.put(presignedUrlResponse.url, renamedFile, config)
                        .then(response => {
                            if (this.$data.shouldSetFileUuidToMessage) {
                                bus.$emit(SET_FILE_ITEM_UUID, {
                                    fileItemUuid: this.fileItemUuid,
                                    count: (presignedUrlResponse.existingCount + index + 1)
                                });
                            }
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
        updateChosenFiles(files) {
            console.log("updateChosenFiles", files);
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
        formattedProgress() {
            return formatSize(progressLoaded) + " / " + formatSize(progressTotal)
        },
    },
    created() {
        bus.on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        bus.on(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
        this.onProgressFunction = throttle(this.onProgressFunction, 100);
    },
    destroyed() {
        bus.off(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.off(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        bus.off(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
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
