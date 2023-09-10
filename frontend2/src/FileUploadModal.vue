<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" :persistent="isLoadingPresignedLinks || fileInputQueueHasElements" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.upload_files')">

                <v-card-text>
                    <v-file-input
                        v-if="showFileInput"
                        :disabled="fileUploadingQueueHasElements"
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

                    <template v-for="item in chatStore.fileUploadingQueue">
                        <v-row no-gutters class="d-flex flex-row my-1" >
                            <v-col class="flex-grow-1 flex-shrink-0">
                                <v-progress-linear
                                    v-if="fileUploadingQueueHasElements"
                                    v-model="item.progress"
                                    color="success"
                                    buffer-value="0"
                                    stream
                                    height="25"
                                >
                                    <span class="inprogress-filename text-truncate">{{ formattedFilename(item) }}</span><v-spacer/>
                                    <span class="inprogress-bytes v-col-auto">{{ formattedProgress(item) }}</span>
                                </v-progress-linear>
                            </v-col>
                            <v-col class="flex-grow-0 flex-shrink-0">
                                <v-btn @click="cancel(item)" class="ml-2 upload-cancel-btn" size="false" variant="plain" :title="$vuetify.locale.t('$vuetify.cancel')">
                                    <v-icon color="red">mdi-cancel</v-icon>
                                </v-btn>
                            </v-col>
                        </v-row>
                    </template>
                </v-card-text>

                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="flat" min-width="0" v-if="messageIdToAttachFiles" @click="onAttachFilesToMessage()" :title="$vuetify.locale.t('$vuetify.attach_files_to_message')"><v-icon size="large">mdi-attachment-plus</v-icon></v-btn>
                    <template v-if="!limitError && fileInputQueueHasElements">
                        <v-btn color="primary" variant="flat" @click="upload()">{{ $vuetify.locale.t('$vuetify.upload') }}</v-btn>
                    </template>
                    <v-btn @click="hideModal()" color="red" variant="flat">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
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
  FILE_UPLOAD_MODAL_START_UPLOADING, INCREMENT_FILE_ITEM_FILE_COUNT, ATTACH_FILES_TO_MESSAGE_MODAL
} from "./bus/bus";
import axios from "axios";
import throttle from "lodash/throttle";
import { formatSize } from "./utils";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";
import {v4 as uuidv4} from "uuid";
const CancelToken = axios.CancelToken;

export default {
    data () {
        return {
            show: false,
            inputFiles: [],
            fileItemUuid: null, // null at first upload, non-nul when user adds files,
            limitError: null,
            showFileInput: false,
            isLoadingPresignedLinks: false,
            shouldSetFileUuidToMessage: false,
            correlationId: null,
            messageIdToAttachFiles: null,
        }
    },
    methods: {
        showModal({showFileInput, fileItemUuid, shouldSetFileUuidToMessage, predefinedFiles, correlationId, messageIdToAttachFiles}) {
            this.$data.show = true;
            this.$data.fileItemUuid = fileItemUuid;
            this.$data.showFileInput = showFileInput;
            this.$data.shouldSetFileUuidToMessage = shouldSetFileUuidToMessage;
            if (predefinedFiles) {
                this.$data.inputFiles = predefinedFiles;
            }
            this.messageIdToAttachFiles = messageIdToAttachFiles;
            this.correlationId = correlationId;
            console.log("Opened FileUploadModal with fileItemUuid=", fileItemUuid, ", shouldSetFileUuidToMessage=", shouldSetFileUuidToMessage, ", predefinedFiles=", predefinedFiles, ", correlationId=", correlationId);
        },
        hideModal() {
            this.$data.show = false;
            this.inputFiles = [];
            this.limitError = null;
            this.showFileInput = false;
            this.$data.isLoadingPresignedLinks = false;
            this.$data.fileItemUuid = null;
            this.$data.shouldSetFileUuidToMessage = false;
            this.correlationId = null;
            this.messageIdToAttachFiles = null;
        },
        onAttachFilesToMessage() {
          bus.emit(ATTACH_FILES_TO_MESSAGE_MODAL, {messageId: this.messageIdToAttachFiles})
          this.hideModal();
        },
        onProgressFunction(add, total, progressReceiver) {
            const progressFunction = (event) => {
                const loaded = add + event.loaded;
                progressReceiver.progress = Math.round((100 * loaded) / total);
                progressReceiver.progressLoaded = loaded;
                progressReceiver.progressTotal = total;
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
            this.$data.isLoadingPresignedLinks = true;

            let totalSize = 0;
            for (const file of this.inputFiles) {
                totalSize += file.size;
            }

            try {
                await this.checkLimits(totalSize)
            } catch (errMsg) {
                return Promise.resolve();
            }

            while (this.fileInputQueueHasElements) {
                const file = this.inputFiles.shift();

                // This 3-step algorithm is made to bypass 2GB file issue in Firefox
                // (it seems there is a timeout inside, because on local machine it works fine)

                // [1/3] init s3's multipart upload
                const response = await axios.put(`/api/storage/${this.chatId}/upload/init`, {
                    fileItemUuid: this.fileItemUuid, // nullable
                    fileSize: file.size,
                    fileName: file.name,
                    correlationId: this.correlationId, // nullable
                })
                console.log("For", file.name, "got init response: ", response.data)

                this.chatStore.appendToFileUploadingQueue({
                    id: uuidv4(),
                    presignedUrls: response.data.presignedUrls,
                    file: file,
                    fileItemUuid: response.data.fileItemUuid,
                    newFileName: response.data.newFileName,
                    key: response.data.key,
                    uploadId: response.data.uploadId,
                    existingCount: response.data.existingCount,
                    progress: 0,
                    progressLoaded: 0,
                    progressTotal: 0,
                    cancelSource: CancelToken.source(),
                    chatId: response.data.chatId,
                    shouldSetFileUuidToMessage: this.$data.shouldSetFileUuidToMessage,
                    chunkSize: response.data.chunkSize,
                })
            }
            this.$data.fileItemUuid = null;
            this.showFileInput = false;
            this.$data.isLoadingPresignedLinks = false;


            console.log("Sending files to storage", this.chatStore.fileUploadingQueue);

            const fileUploadingQueueCopy = [...this.chatStore.fileUploadingQueue];
            for (const fileToUpload of fileUploadingQueueCopy) {
                try {
                    // renaming a file
                    const formData = new FormData();
                    const partName = "File";
                    formData.append(partName, fileToUpload.file, fileToUpload.newFileName);
                    const renamedFile = formData.get(partName);

                    const config = {
                        cancelToken: fileToUpload.cancelSource.token
                    }

                    const chunkSize = fileToUpload.chunkSize;
                    const uploadResults = [];
                    // [2/3] upload parts by s3's presigned links
                    for (const presignedUrlObj of fileToUpload.presignedUrls) {
                      const partNumber = presignedUrlObj.partNumber; // starts from 1
                      const start = (partNumber - 1) * chunkSize;
                      const end = partNumber * chunkSize;
                      console.log("Will send part", presignedUrlObj, start, end, chunkSize);
                      const blob = renamedFile.slice(start, end);

                      const childConfig = {
                        ...config,
                        onUploadProgress: this.onProgressFunction(start, fileToUpload.file.size, fileToUpload),
                      };

                      const res = await axios.put(presignedUrlObj.url, blob, childConfig);
                      uploadResults.push({etag: JSON.parse(res.headers.etag), partNumber: partNumber});
                    }

                    // [3/3] concatenate parts
                    await axios.put(`/api/storage/${fileToUpload.chatId}/upload/finish`, {
                      key: fileToUpload.key,
                      parts: uploadResults,
                      uploadId: fileToUpload.uploadId
                    });

                    if (fileToUpload.shouldSetFileUuidToMessage) {
                      bus.emit(SET_FILE_ITEM_UUID, {
                        fileItemUuid: fileToUpload.fileItemUuid,
                        chatId: fileToUpload.chatId,
                      });
                      bus.emit(INCREMENT_FILE_ITEM_FILE_COUNT, {
                        chatId: fileToUpload.chatId,
                      });
                    }
                } catch(thrown) {
                    if (axios.isCancel(thrown)) {
                        console.log('Request canceled', thrown.message);
                    } else {
                        console.warn('Request failed', thrown);
                    }
                } finally {
                    this.chatStore.removeFromFileUploadingQueue(fileToUpload.id);
                }
            }
            this.hideModal();
            return Promise.resolve();
        },
        cancel(item) {
            item.cancelSource.cancel()
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
        fileUploadingQueueHasElements() {
            return !!this.chatStore.fileUploadingQueue.length
        },
        fileInputQueueHasElements() {
            return !!this.inputFiles.length
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
