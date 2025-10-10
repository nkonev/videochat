<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" :persistent="checkingLimitsStep || isLoadingPresignedLinks || fileInputQueueHasElements || fileUploadingQueueHasElements" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.upload_files')">
                <v-progress-linear
                    :active="checkingLimitsStep || isLoadingPresignedLinks"
                    indeterminate
                    absolute
                    bottom
                    color="primary"
                ></v-progress-linear>

                <v-card-text>
                    <v-file-input
                        v-if="showFileInput"
                        :disabled="checkingLimitsStep || isLoadingPresignedLinks || fileUploadingQueueHasElements"
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
                    <v-checkbox
                        v-if="shouldShowSendMessageAfterMediaInsert()"
                        density="comfortable"
                        color="primary"
                        hide-details
                        v-model="chatStore.sendMessageAfterUploadsUploaded"
                        :label="$vuetify.locale.t('$vuetify.send_message_after_media_insert')"
                        :title="$vuetify.locale.t('$vuetify.send_message_after_media_insert_description')"
                    ></v-checkbox>
                    <v-btn variant="outlined" min-width="0" v-if="shouldShowAttachExistingFilesToMessage" @click="onAttachFilesToMessage()" :title="$vuetify.locale.t('$vuetify.attach_files_to_message')"><v-icon size="large">mdi-attachment-plus</v-icon></v-btn>
                    <template v-if="!limitError && fileInputQueueHasElements">
                        <v-btn color="primary" variant="flat" @click="upload()" :loading="checkingLimitsStep" :disabled="checkingLimitsStep">{{ $vuetify.locale.t('$vuetify.upload') }}</v-btn>
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
    MESSAGE_EDIT_SET_FILE_ITEM_UUID,
    FILE_UPLOAD_MODAL_START_UPLOADING,
    ATTACH_FILES_TO_MESSAGE_MODAL,
} from "./bus/bus";
import axios from "axios";
import throttle from "lodash/throttle";
import {formatSize, hasLength, renameFilePart} from "./utils";
import {mapStores} from "pinia";
import {fileUploadingSessionTypeMessageEdit, fileUploadingSessionTypeMedia, useChatStore} from "@/store/chatStore";
import {v4 as uuidv4} from "uuid";
import {retry} from "@lifeomic/attempt";
const CancelToken = axios.CancelToken;

export default {
    data () {
        return {
            show: false,
            inputFiles: [],
            // we cannot remove it, because all the attempts lead to the bugs which shows
            // that it's better to have a dedicated copy of it per view than have one and have unexpected side effects
            fileItemUuid: null,
            limitError: null,
            showFileInput: false,
            isLoadingPresignedLinks: false,
            shouldSetFileUuidToMessage: false,
            messageIdToAttachFiles: null,
            shouldAddDateToTheFilename: null,
            checkingLimitsStep: false,
            isMessageRecording: null,
            correlationId: null,
        }
    },
    methods: {
        showModal({
              showFileInput,
              fileItemUuid,
              shouldSetFileUuidToMessage,
              predefinedFiles,
              messageIdToAttachFiles,
              shouldAddDateToTheFilename,
              fileUploadingSessionType,
              isMessageRecording,
              correlationId,
            }) {
            this.$data.show = true;
            this.$data.fileItemUuid = fileItemUuid;
            this.$data.showFileInput = showFileInput;
            this.$data.shouldSetFileUuidToMessage = shouldSetFileUuidToMessage;
            if (predefinedFiles) {
                this.$data.inputFiles = predefinedFiles;
            }
            this.messageIdToAttachFiles = messageIdToAttachFiles;
            this.shouldAddDateToTheFilename = shouldAddDateToTheFilename;
            if (!this.chatStore.fileUploadingQueueHasElements()) { // there is no prev active uploading
                this.chatStore.setFileUploadingSessionType(fileUploadingSessionType)
            }
            this.isMessageRecording = isMessageRecording;
            this.correlationId = correlationId;
            console.log("Opened FileUploadModal with fileItemUuid=", fileItemUuid, ", correlationId=", correlationId, ", shouldSetFileUuidToMessage=", shouldSetFileUuidToMessage, ", predefinedFiles=", predefinedFiles, "shouldAddDateToTheFilename=", shouldAddDateToTheFilename, ", fileUploadingSessionType=", fileUploadingSessionType, ", isMessageRecording=", isMessageRecording);
        },
        hideModal() {
            this.$data.show = false;
            this.inputFiles = [];
            this.limitError = null;
            this.showFileInput = false;
            this.$data.isLoadingPresignedLinks = false;
            this.checkingLimitsStep = false;
            this.$data.fileItemUuid = null;
            this.$data.shouldSetFileUuidToMessage = false;
            this.messageIdToAttachFiles = null;
            this.shouldAddDateToTheFilename = null;
            this.isMessageRecording = null;
            this.correlationId = null;
        },
        onAttachFilesToMessage() {
          bus.emit(ATTACH_FILES_TO_MESSAGE_MODAL, {messageId: this.messageIdToAttachFiles})
          this.hideModal();
        },
        fileUploadOverallProgress() {
            let loaded = 0;
            let total = 0;
            for (const item of this.chatStore.fileUploadingQueue) {
                loaded += item.progressLoaded;
                total += item.progressTotal;
            }
            return Math.round((100 * loaded) / total)
        },
        onProgressFunction(add, total, progressReceiver) {
            const progressFunction = (event) => {
                const loaded = add + event.loaded;
                progressReceiver.progress = Math.round((100 * loaded) / total); // in percents
                progressReceiver.progressLoaded = loaded; // in *bytes

                // calculate total
                this.chatStore.fileUploadOverallProgress = this.fileUploadOverallProgress();
            }
            return throttle(progressFunction, 200)
        },
        checkLimits(totalSize, chatId) {
            return axios.get(`/api/storage/${chatId}/file`, { params: {
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
            const chatId = this.chatId; // use local variable (cache) to have an ability to switch chat during file uploading

            this.$data.isLoadingPresignedLinks = true;
            this.checkingLimitsStep = true;

            let totalSize = 0;
            for (const file of this.inputFiles) {
                totalSize += file.size;
            }

            try {
                await this.checkLimits(totalSize, chatId)
            } catch (errMsg) {
                this.$data.isLoadingPresignedLinks = false;
                this.checkingLimitsStep = false;
                return Promise.resolve();
            }
            this.checkingLimitsStep = false;

            while (this.fileInputQueueHasElements) {
                const file = this.inputFiles.shift();
                if (file.isProcessing) {
                    continue
                }
                file.isProcessing = true;

                // This 3-step algorithm is made to bypass 2GB file issue in Firefox
                // (it seems there is a timeout inside, because on local machine it works fine)

                let response;
                try {
                    // [1/3] init s3's multipart upload
                    response = await axios.put(`/api/storage/${chatId}/upload/init`, {
                        fileItemUuid: this.fileItemUuid, // nullable
                        fileSize: file.size,
                        fileName: file.name,
                        shouldAddDateToTheFilename: this.shouldAddDateToTheFilename, // nullable
                        isMessageRecording: this.isMessageRecording, // nullable
                    }, {
                      headers: {
                        "X-CorrelationId": this.correlationId,
                      }
                    })
                    console.log("For", file.name, "got init response: ", response.data)
                } catch(thrown) {
                    this.cleanOnError(thrown);
                    continue
                }

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
                    progressTotal: file.size,
                    cancelSource: CancelToken.source(),
                    chatId: response.data.chatId,
                    shouldSetFileUuidToMessage: this.$data.shouldSetFileUuidToMessage,
                    chunkSize: response.data.chunkSize,
                    previewable: response.data.previewable,
                })
                this.$data.fileItemUuid = response.data.fileItemUuid;
            }
            this.$data.fileItemUuid = null;
            this.correlationId = null;
            this.showFileInput = false;
            this.$data.isLoadingPresignedLinks = false;
            this.chatStore.sendMessageAfterMediaNumFiles = this.chatStore.fileUploadingQueue.filter(f => f.previewable).length
            this.chatStore.sendMessageAfterNumFiles = this.chatStore.fileUploadingQueue.length

            console.log("Sending files to storage", this.chatStore.fileUploadingQueue);

            const fileUploadingQueueCopy = [...this.chatStore.fileUploadingQueue];
            for (const fileToUpload of fileUploadingQueueCopy) {
                if (fileToUpload.isProcessing || fileToUpload.finished) {
                    continue
                }
                fileToUpload.isProcessing = true;
                fileToUpload.finished = false;

                try {
                    // renaming a file
                    const renamedFile = renameFilePart(fileToUpload.file, fileToUpload.newFileName);

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

                      this.setSendMessageAfterUploadFileItemUuidInNeed(fileToUpload.fileItemUuid);

                      const childConfig = {
                        ...config,
                        onUploadProgress: this.onProgressFunction(start, fileToUpload.file.size, fileToUpload),
                      };

                      const retryOptions = {
                        delay: 2000,
                        maxAttempts: 10,
                        handleError (err, context) {
                          if (axios.isCancel(err)) {
                            // We should abort because error indicates that request is not retryable
                            context.abort();
                          }
                        }
                      };

                      const res = await retry((context) => {
                        const blob = renamedFile.slice(start, end);
                        return axios.put(presignedUrlObj.url, blob, childConfig)
                          .catch((e) => {
                            if (axios.isCancel(e)) {
                              throw e
                            }
                            this.setWarning("An error during uploading '" + renamedFile.name + "', retrying, attempt " + (context.attemptNum + 1) + " / " + retryOptions.maxAttempts);
                            console.warn("Error", e);
                            throw e
                        })
                      }, retryOptions);

                      this.setSendMessageAfterUploadFileItemUuidInNeed(fileToUpload.fileItemUuid);

                      uploadResults.push({etag: JSON.parse(res.headers.etag), partNumber: partNumber});
                    }

                    // in order to propagate it back to MessageEdit, TipTapEditor and others
                    if (fileToUpload.shouldSetFileUuidToMessage) {
                      bus.emit(MESSAGE_EDIT_SET_FILE_ITEM_UUID, {
                        fileItemUuid: fileToUpload.fileItemUuid,
                        chatId: fileToUpload.chatId,
                      });
                    }

                    // [3/3] concatenate parts
                    await axios.put(`/api/storage/${fileToUpload.chatId}/upload/finish`, {
                      key: fileToUpload.key,
                      parts: uploadResults,
                      uploadId: fileToUpload.uploadId
                    });

                } catch(thrown) {
                    this.cleanOnError(thrown)
                } finally {
                    fileToUpload.finished = true;
                }
            }

            if (this.chatStore.fileUploadingQueue.length == this.chatStore.fileUploadingQueue.filter((item) => item.finished).length) {
                this.chatStore.cleanFileUploadingQueue();
            }
            this.hideModal();
            return Promise.resolve();
        },
        shouldShowSendMessageAfterMediaInsert() {
            return this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMessageEdit || this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMedia
        },
        cancel(item) {
            item.cancelSource.cancel();
            this.chatStore.sendMessageAfterUploadsUploaded = false;
            this.chatStore.sendMessageAfterUploadFileItemUuid = null;
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
                this.checkLimits(totalSize, this.chatId);
            }
        },
        formattedProgress(progressReceiver) {
            return formatSize(progressReceiver.progressLoaded) + " / " + formatSize(progressReceiver.progressTotal)
        },
        formattedFilename(progressReceiver) {
            return progressReceiver.file.name
        },
        cleanOnError(thrown) {
          if (axios.isCancel(thrown)) {
            console.log('Request canceled', thrown.message);
          } else {
            console.warn('Request failed', thrown);
          }

          this.limitError = null;
          this.showFileInput = true;
          this.$data.isLoadingPresignedLinks = false;
          this.checkingLimitsStep = false;
        },

        setSendMessageAfterUploadFileItemUuidInNeed(f) {
          if (this.chatStore.sendMessageAfterUploadsUploaded) {
            this.chatStore.sendMessageAfterUploadFileItemUuid = f;
          }
        },
    },
    computed: {
        ...mapStores(useChatStore),
        chatId() {
            return this.$route.params.id
        },
        fileUploadingQueueHasElements() {
            return this.chatStore.fileUploadingQueueHasElements()
        },
        fileInputQueueHasElements() {
            return !!this.inputFiles.length
        },
        shouldShowAttachExistingFilesToMessage() {
          return this.messageIdToAttachFiles && !this.fileUploadingQueueHasElements && !this.fileInputQueueHasElements
        }
    },
    mounted() {
        bus.on(OPEN_FILE_UPLOAD_MODAL, this.showModal);
        bus.on(CLOSE_FILE_UPLOAD_MODAL, this.hideModal);
        bus.on(FILE_UPLOAD_MODAL_START_UPLOADING, this.upload);
    },
    beforeUnmount() {
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
