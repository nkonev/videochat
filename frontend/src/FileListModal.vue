<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" min-height="400" scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title class="d-flex align-center ml-2">
                    {{ fileItemUuid ? $vuetify.locale.t('$vuetify.attached_message_files') : $vuetify.locale.t('$vuetify.attached_chat_files') }}
                    <v-text-field class="ml-4" variant="outlined" density="compact" prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="$vuetify.locale.t('$vuetify.search_by_files')" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
                </v-card-title>

                <v-card-text class="py-2">
                    <v-row v-if="!loading">
                        <template v-if="dto.count > 0">
                            <v-col
                                v-for="item in dto.files"
                                :key="item.id"
                                :cols="6"
                            >
                                <v-card>
                                    <v-img
                                        :src="item.previewUrl"
                                        class="white--text align-end"
                                        gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
                                        height="200px"
                                    >
                                        <v-container class="file-info-title ma-0 pa-0">
                                        <v-card-title class="pb-1">
                                            <a :href="item.url" download class="breaks download-link text-white">{{item.filename}}</a>
                                        </v-card-title>
                                        <v-card-subtitle class="text-white pb-2 no-opacity text-wrap">
                                            {{ formattedSize(item.size) }}
                                            <span v-if="item.owner"> {{ $vuetify.locale.t('$vuetify.files_by') }} {{item.owner.login}}</span>
                                            <span> {{$vuetify.locale.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                            <a v-if="item.publicUrl" :href="item.publicUrl" target="_blank" class="colored-link">
                                                {{ $vuetify.locale.t('$vuetify.files_public_url') }}
                                            </a>
                                        </v-card-subtitle>
                                        </v-container>
                                    </v-img>
                                    <v-card-actions>
                                        <v-spacer></v-spacer>
                                        <v-btn size="medium" v-if="item.canShowAsImage" @click="fireShowImage(item)" :title="$vuetify.locale.t('$vuetify.view')"><v-icon size="large">mdi-image</v-icon></v-btn>

                                        <v-btn size="medium" v-if="item.canPlayAsVideo" @click="fireVideoPlay(item)" :title="$vuetify.locale.t('$vuetify.play')"><v-icon size="large">mdi-play</v-icon></v-btn>

                                        <v-btn size="medium" v-if="item.canEdit" @click="fireEdit(item)" :title="$vuetify.locale.t('$vuetify.edit')"><v-icon size="large">mdi-pencil</v-icon></v-btn>

                                        <v-btn size="medium" v-if="item.canShare" @click="shareFile(item, !item.publicUrl)">
                                            <v-icon color="primary" size="large" dark :title="item.publicUrl ? $vuetify.locale.t('$vuetify.unshare_file') : $vuetify.locale.t('$vuetify.share_file')">{{ item.publicUrl ? 'mdi-lock' : 'mdi-export'}}</v-icon>
                                        </v-btn>

                                        <v-btn size="medium" v-if="item.canDelete" @click="deleteFile(item)">
                                            <v-icon color="red" size="large" dark :title="$vuetify.locale.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                                        </v-btn>
                                    </v-card-actions>
                                </v-card>
                            </v-col>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-row>
                    <v-progress-circular
                        class="my-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">

                  <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                  <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                    <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0">
                      <v-pagination
                          variant="elevated"
                          active-color="primary"
                          density="comfortable"
                          v-if="shouldShowPagination"
                          v-model="page"
                          :length="pagesCount"
                          :total-visible="7"
                      ></v-pagination>
                    </v-col>
                    <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-0 flex-shrink-0 align-self-end">
                      <v-btn variant="outlined" min-width="0" v-if="messageIdToDetachFiles" @click="onDetachFilesFromMessage()" :title="$vuetify.locale.t('$vuetify.detach_files_from_message')"><v-icon size="large">mdi-attachment-minus</v-icon></v-btn>
                      <v-btn variant="flat" color="primary" @click="openUploadModal()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.locale.t('$vuetify.upload') }}</v-btn>
                      <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                    </v-col>
                  </v-row>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
  CLOSE_SIMPLE_MODAL,
  PREVIEW_CREATED,
  OPEN_FILE_UPLOAD_MODAL,
  OPEN_SIMPLE_MODAL,
  OPEN_TEXT_EDIT_MODAL,
  OPEN_VIEW_FILES_DIALOG,
  SET_FILE_ITEM_UUID,
  FILE_CREATED,
  FILE_REMOVED,
  PLAYER_MODAL,
  FILE_UPDATED, LOAD_FILES_COUNT
} from "./bus/bus";
import axios from "axios";
import {
    getHumanReadableDate,
    replaceInArray,
    formatSize,
    hasLength,
    findIndex,
    replaceOrPrepend
} from "./utils";
import debounce from "lodash/debounce";
import {mapStores} from "pinia";
import {useChatStore} from "@/store/chatStore";

const firstPage = 1;
const pageSize = 20;

const dtoFactory = () => {return {files: [], count: 0} };

export default {
    data () {
        return {
            show: false,
            messageIdToDetachFiles: null,
            dto: dtoFactory(),
            fileItemUuid: null,
            loading: false,
            messageEditing: false,
            page: firstPage,
            searchString: null,
        }
    },
    computed: {
        pagesCount() {
            const count = Math.ceil(this.dto.count / pageSize);
            // console.debug("Calc pages count", count);
            return count;
        },
        shouldShowPagination() {
            return this.dto != null && this.dto.files && this.dto.count > pageSize
        },
        chatId() {
            return this.$route.params.id
        },
        ...mapStores(useChatStore),
    },

    methods: {
        showModal({fileItemUuid, messageEditing, messageIdToDetachFiles}) {
            console.log("Opening files modal, fileItemUuid=", fileItemUuid);
            this.fileItemUuid = fileItemUuid;
            this.show = true;
            this.messageIdToDetachFiles = messageIdToDetachFiles;
            this.messageEditing = messageEditing;
            this.updateFiles();
        },
        translatePage() {
            return this.page - 1;
        },
        updateFiles() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get(`/api/storage/${this.chatId}`, {
                params: {
                    page: this.translatePage(),
                    size: pageSize,
                    fileItemUuid : this.fileItemUuid ? this.fileItemUuid : '',
                    searchString: this.searchString
                },
            })
                .then((response) => {
                    this.dto = response.data;
                })
                .finally(() => {
                    this.loading = false;
                })
        },
        closeModal() {
            this.show = false;
            this.messageIdToDetachFiles = null;
            this.fileItemUuid = null;
            this.messageEditing = false;
            this.page = firstPage;
            this.searchString = null;
            this.dto = dtoFactory();
        },
        doSearch(){
            this.updateFiles();
        },
        openUploadModal() {
            bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: this.fileItemUuid, shouldSetFileUuidToMessage: this.messageEditing});
        },
        onDetachFilesFromMessage () {
          axios.put(`/api/chat/`+this.chatId+'/message/file-item-uuid', {
            messageId: this.messageIdToDetachFiles,
            fileItemUuid: null
          }).then(()=>{
            bus.emit(SET_FILE_ITEM_UUID, {fileItemUuid: null, chatId: this.chatId});
            bus.emit(LOAD_FILES_COUNT);
            this.closeModal();
          })
        },
        deleteFile(dto) {
            bus.emit(OPEN_SIMPLE_MODAL, {
                buttonName: this.$vuetify.locale.t('$vuetify.delete_btn'),
                title: this.$vuetify.locale.t('$vuetify.delete_file_title'),
                text: this.$vuetify.locale.t('$vuetify.delete_file_text', dto.filename),
                actionFunction: (that)=> {
                    that.loading = true;
                    axios.delete(`/api/storage/${this.chatId}/file`, {
                        data: {id: dto.id},
                        params: {
                            page: this.translatePage(),
                            size: pageSize,
                            fileItemUuid : this.fileItemUuid ? this.fileItemUuid : ''
                        }
                    })
                    .then((response) => {
                        if (this.$data.messageEditing) {
                            bus.emit(LOAD_FILES_COUNT);
                        }

                        bus.emit(CLOSE_SIMPLE_MODAL);
                    }).finally(()=>{
                      that.loading = false;
                    })
                }
            });
        },
        shareFile(dto, share) {
            axios.put(`/api/storage/publish/file`, {id: dto.id, public: share})
        },
        fireEdit(dto) {
            bus.emit(OPEN_TEXT_EDIT_MODAL, {fileInfoDto: dto, chatId: this.chatId, fileItemUuid: this.fileItemUuid});
        },
        fireVideoPlay(dto) {
            bus.emit(PLAYER_MODAL, dto);
        },
        fireShowImage(dto) {
            bus.emit(PLAYER_MODAL, dto);
        },
        getDate(item) {
            return getHumanReadableDate(item.lastModified)
        },
        hasSearchString() {
            return hasLength(this.searchString)
        },
        resetInput() {
            this.searchString = null;
        },
        onPreviewCreated(dto) {
            if (!this.show) {
                return
            }

            console.log("Replacing preview", dto);
            for (const fileItem of this.dto.files) {
                if (fileItem.id == dto.id) {
                    fileItem.previewUrl = dto.previewUrl;
                    break
                }
            }
        },
        removeItem(dto) {
            console.log("Removing item", dto);
            const idxToRemove = findIndex(this.dto.files, dto);
            this.dto.files.splice(idxToRemove, 1);
        },
        replaceItem(dto) {
            console.log("Replacing item", dto);
            replaceOrPrepend(this.dto.files, [dto]);
        },
        onFileCreated(dto) {
            console.log("onFileCreated", dto);
            if (!this.show) {
                return
            }
            if (!hasLength(this.fileItemUuid) || dto.fileItemUuid == this.fileItemUuid) {
                if (!hasLength(this.fileItemUuid)) {
                    this.dto.count = dto.count;
                }
                this.replaceItem(dto.fileInfoDto);
            }
        },
        onFileUpdated(dto) {
            console.log("onFileUpdated", dto);
            if (!this.show) {
                return
            }
            if (!hasLength(this.fileItemUuid) || dto.fileItemUuid == this.fileItemUuid) {
                if (!hasLength(this.fileItemUuid)) {
                  this.dto.count = dto.count;
                }
                this.replaceItem(dto.fileInfoDto);
            }
        },
        onFileRemoved(dto) {
            if (!this.show) {
                return
            }
            if (!hasLength(this.fileItemUuid)) {
                this.dto.count = dto.count;
            }
            this.removeItem(dto.fileInfoDto);
        },
        formattedSize(size) {
            return formatSize(size)
        },
    },
    watch: {
        page(newValue) {
            if (this.show) {
                console.debug("SettingNewPage", newValue);
                this.dto = dtoFactory();
                this.updateFiles();
            }
        },
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
        searchString(searchString) {
            this.doSearch();
        },
    },
    created() {
        this.doSearch = debounce(this.doSearch, 700);
    },
    mounted() {
      bus.on(OPEN_VIEW_FILES_DIALOG, this.showModal);
      bus.on(PREVIEW_CREATED, this.onPreviewCreated);
      bus.on(FILE_CREATED, this.onFileCreated);
      bus.on(FILE_UPDATED, this.onFileUpdated);
      bus.on(FILE_REMOVED, this.onFileRemoved);
    },
    beforeUnmount() {
        bus.off(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.off(PREVIEW_CREATED, this.onPreviewCreated);
        bus.off(FILE_CREATED, this.onFileCreated);
        bus.off(FILE_UPDATED, this.onFileUpdated);
        bus.off(FILE_REMOVED, this.onFileRemoved);
    },
}
</script>

<style lang="stylus" scoped>
@import "constants.styl"
.no-opacity {
  opacity 1
}
.breaks {
  white-space: break-spaces;
}
.download-link {
    text-decoration none
}
.file-info-title {
    background rgba(0, 0, 0, 0.5);
}

</style>
