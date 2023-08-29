<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="400" :persistent="uploading || inputFiles.length > 0">
            <v-card :title="$vuetify.locale.t('$vuetify.upload_files')">

                <v-container>
                    <v-file-input
                        :disabled="uploading"
                        :model-value="inputFiles"
                        counter
                        multiple
                        show-size
                        chips
                        single-line
                        @update:modelValue="updateChosenFiles"
                        :error-messages="limitError ? [limitError] : []"
                        variant="underlined"
                    ></v-file-input>

                    <v-divider/>

                    <template v-for="item in chatStore.fileUploadingQueue">
                        <v-progress-linear
                            class="mt-2"
                            v-if="uploading"
                            v-model="item.progress"
                            color="success"
                            buffer-value="0"
                            stream
                            height="25"
                        >
                          <span class="inprogress-filename">{{ formattedFilename(item) }}</span><v-spacer/><span class="inprogress-bytes">{{ formattedProgress(item) }}</span>
                        </v-progress-linear>
                    </template>
                </v-container>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <template v-if="!limitError && inputFiles.length > 0">
                        <v-btn v-if="!uploading" color="primary" variant="flat" @click="upload()">{{ $vuetify.locale.t('$vuetify.upload') }}</v-btn>
                        <v-btn v-else @click="cancel()" variant="outlined">{{ $vuetify.locale.t('$vuetify.cancel') }}</v-btn>
                    </template>
                    <v-btn @click="hideModal()" :disabled="uploading" color="red" variant="flat">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
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
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
const CancelToken = axios.CancelToken;

export default {
    data () {
        return {
            show: false,
            inputFiles: [],
            fileItemUuid: null, // null at first upload, non-nul when user adds files,
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
                this.$data.inputFiles = predefinedFiles;
                this.$data.filesWerePredefined = true;
            }
            this.correlationId = correlationId;
            console.log("Opened FileUploadModal with fileItemUuid=", fileItemUuid, ", shouldSetFileUuidToMessage=", shouldSetFileUuidToMessage, ", predefinedFiles=", predefinedFiles, ", correlationId=", correlationId);
        },
        hideModal() {
            this.$data.show = false;
            this.inputFiles = [];
            this.cancelSource = null; // todo per file
            this.limitError = null;
            this.$data.fileItemUuid = null;
            this.$data.shouldSetFileUuidToMessage = false;
            this.$data.filesWerePredefined = false;
            this.correlationId = null;
        },
        onProgressFunction(progressReceiver) {
            const progressFunction = (event) => {
                progressReceiver.progress = Math.round((100 * event.loaded) / event.total);
                progressReceiver.progressLoaded = (event.loaded);
                progressReceiver.progressTotal = (event.total);
            }
            return throttle(progressFunction, 100)
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
            let totalSize = 0;
            for (const file of this.inputFiles) {
                totalSize += file.size;
            }

            try {
                await this.checkLimits(totalSize)
            } catch (errMsg) {
                return Promise.resolve();
            }

            for (const file of this.inputFiles) {
                this.chatStore.appendToFileUploadingQueue({file: file, progress: 50, progressLoaded: 0, progressTotal: 0})
            }







            // this.cancelSource = CancelToken.source();
            // const config = {
            //     // headers: { 'content-type': 'multipart/form-data' },
            //     onUploadProgress: this.onProgressFunction(item),
            //     cancelToken: this.cancelSource.token
            // }
            // console.log("Sending file to storage");
            //
            // const urlResponses = [];
            // for (const file of this.inputFiles) {
            //     const response = await axios.put(`/api/storage/${this.chatId}/url`, {
            //         fileItemUuid: this.fileItemUuid, // nullable
            //         fileSize: file.size,
            //         fileName: file.name,
            //         correlationId: this.correlationId, // nullable
            //     })
            //     this.fileItemUuid = response.data.fileItemUuid;
            //     urlResponses.push({
            //         url: response.data.url,
            //         file: file,
            //         newFileName: response.data.newFileName,
            //         existingCount: response.data.existingCount,
            //     });
            // }
            //
            // for (const [index, presignedUrlResponse] of urlResponses.entries()) {
            //     try {
            //         const formData = new FormData();
            //         formData.append('File', presignedUrlResponse.file, presignedUrlResponse.newFileName);
            //         const renamedFile = formData.get('File');
            //
            //         await axios.put(presignedUrlResponse.url, renamedFile, config)
            //             .then(response => {
            //                 if (this.$data.shouldSetFileUuidToMessage) {
            //                     bus.emit(SET_FILE_ITEM_UUID, {
            //                         fileItemUuid: this.fileItemUuid,
            //                         count: (presignedUrlResponse.existingCount + index + 1)
            //                     });
            //                 }
            //                 return response;
            //             })
            //     } catch(thrown) {
            //         if (axios.isCancel(thrown)) {
            //             console.log('Request canceled', thrown.message);
            //             break
            //         } else {
            //             return Promise.reject(thrown);
            //         }
            //     }
            // }
            // this.hideModal();
            return Promise.resolve();
        },
        cancel() {
            this.cancelSource.cancel()
        },
        updateChosenFiles(files) {
            console.log("updateChosenFiles", files);
            this.inputFiles = [...files];
            this.limitError = null;
            let totalSize = 0;
            for (const file of this.inputFiles) {
                totalSize += file.size;
            }
            if (totalSize > 0) {
                this.checkLimits(totalSize);
            }
        },
        formattedProgress(progressReceiver) {
            return formatSize(progressReceiver.progressLoaded) + " / " + formatSize(progressReceiver.progressTotal)
        },
        formattedFilename(progressReceiver) {
            return progressReceiver.file.name
        },
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        uploading() {
            return !!this.chatStore.fileUploadingQueue.length
        }
    },
    created() {
        bus.on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        bus.on(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
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

<style scoped lang="stylus">
    .inprogress-filename {
        padding-left 0.4em
    }
    .inprogress-bytes {
        font-weight bold
    }
</style>
