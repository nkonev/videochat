<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" scrollable :persistent="hasSearchString()">
            <v-card>
                <v-card-title>
                    {{ fileItemUuid ? $vuetify.lang.t('$vuetify.attached_message_files') : $vuetify.lang.t('$vuetify.attached_chat_files') }}
                    <v-text-field class="ml-4 pt-0 mt-0" prepend-icon="mdi-magnify" hide-details single-line v-model="searchString" :label="$vuetify.lang.t('$vuetify.search_by_files')" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput"></v-text-field>
                </v-card-title>

                <v-card-text>
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
                                        <v-card-title>
                                            <a :href="item.url" target="_blank" class="download-link">{{item.filename}}</a>
                                        </v-card-title>
                                        <v-card-subtitle>
                                            {{ item.size | formatSizeFilter }}
                                            <span v-if="item.owner"> {{ $vuetify.lang.t('$vuetify.files_by') }} {{item.owner.login}}</span>
                                            <span> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                            <a v-if="item.publicUrl" :href="item.publicUrl" target="_blank">
                                                {{ $vuetify.lang.t('$vuetify.files_public_url') }}
                                            </a>
                                        </v-card-subtitle>
                                        </v-container>
                                    </v-img>
                                    <v-card-actions>
                                        <v-spacer></v-spacer>

                                        <v-btn icon v-if="item.canEdit" @click="fireEdit(item)" :title="$vuetify.lang.t('$vuetify.edit')"><v-icon>mdi-pencil</v-icon></v-btn>

                                        <v-btn icon v-if="item.canShare">
                                            <v-icon color="primary" @click="shareFile(item, !item.publicUrl)" dark :title="item.publicUrl ? $vuetify.lang.t('$vuetify.unshare_file') : $vuetify.lang.t('$vuetify.share_file')">{{ item.publicUrl ? 'mdi-lock' : 'mdi-export'}}</v-icon>
                                        </v-btn>

                                        <v-btn icon v-if="item.canDelete">
                                            <v-icon color="error" @click="deleteFile(item)" dark :title="$vuetify.lang.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                                        </v-btn>
                                    </v-card-actions>
                                </v-card>
                            </v-col>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-row>
                    <v-progress-circular
                        class="ma-4 pa-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="filePage"
                        :length="filePagesCount"
                    ></v-pagination>
                    <v-spacer></v-spacer>
                    <v-btn color="primary" class="mr-4 my-1" @click="openUploadModal()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.lang.t('$vuetify.upload') }}</v-btn>
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    CLOSE_SIMPLE_MODAL, FILE_UPLOADED, OPEN_FILE_UPLOAD_MODAL,
    OPEN_SIMPLE_MODAL, OPEN_TEXT_EDIT_MODAL,
    OPEN_VIEW_FILES_DIALOG, SET_FILE_ITEM_UUID, UPDATE_VIEW_FILES_DIALOG
} from "./bus";
import {mapGetters} from "vuex";
import {GET_USER} from "./store";
import axios from "axios";
import {getHumanReadableDate, replaceInArray, formatSize, hasLength} from "./utils";
import debounce from "lodash/debounce";

const firstPage = 1;
const pageSize = 20;

const dtoFactory = () => {return {files: []} };

export default {
    data () {
        return {
            show: false,
            dto: dtoFactory(),
            chatId: null,
            fileItemUuid: null,
            loading: false,
            messageEditing: false,
            filePage: firstPage,
            searchString: null,
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
        filePagesCount() {
            const count = Math.ceil(this.dto.count / pageSize);
            console.debug("Calc pages count", count);
            return count;
        },
        shouldShowPagination() {
            return this.dto != null && this.dto.files && this.dto.count > pageSize
        }
    },

    methods: {
        showModal({chatId, fileItemUuid, messageEditing}) {
            console.log("Opening files modal, chatId=", chatId, ", fileItemUuid=", fileItemUuid);
            this.chatId = chatId;
            this.fileItemUuid = fileItemUuid;
            this.show = true;
            this.messageEditing = messageEditing;
            this.updateFiles();
        },
        translatePage() {
            return this.filePage - 1;
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
            this.chatId = null;
            this.fileItemUuid = null;
            this.messageEditing = false;
            this.filePage = firstPage;
            this.searchString = null;
            this.dto = dtoFactory();
        },
        doSearch(){
            this.updateFiles();
        },
        openUploadModal() {
            bus.$emit(OPEN_FILE_UPLOAD_MODAL, this.fileItemUuid, this.messageEditing);
        },
        deleteFile(dto) {
            bus.$emit(OPEN_SIMPLE_MODAL, {
                buttonName: this.$vuetify.lang.t('$vuetify.delete_btn'),
                title: this.$vuetify.lang.t('$vuetify.delete_file_title'),
                text: this.$vuetify.lang.t('$vuetify.delete_file_text', dto.filename),
                actionFunction: ()=> {
                    axios.delete(`/api/storage/${this.chatId}/file`, {
                        data: {id: dto.id},
                        params: {
                            page: this.translatePage(),
                            size: pageSize,
                            fileItemUuid : this.fileItemUuid ? this.fileItemUuid : ''
                        }
                    })
                        .then((response) => {
                            this.dto = response.data;
                            if (this.$data.messageEditing) {
                                bus.$emit(SET_FILE_ITEM_UUID, {fileItemUuid: this.fileItemUuid, count: response.data.count});
                            }

                            if (this.dto.count == 0) {
                                if (this.filePage > firstPage) {
                                    this.filePage--;
                                    this.updateFiles();
                                } else {
                                    this.closeModal();
                                }
                            }
                            bus.$emit(CLOSE_SIMPLE_MODAL);
                        })
                }
            });
        },
        shareFile(dto, share) {
            axios.put(`/api/storage/publish/file`, {id: dto.id, public: share})
                .then((response) => {
                  replaceInArray(this.dto.files, response.data);
                  this.$nextTick(()=>{
                      this.$forceUpdate();
                  })
                })
        },
        fireEdit(dto) {
            bus.$emit(OPEN_TEXT_EDIT_MODAL, {fileInfoDto: dto, chatId: this.chatId, fileItemUuid: this.fileItemUuid});
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
        onFileUploaded(dto) {
            for (const fileItem of this.dto.files) {
                if (fileItem.id == dto.id) {
                    fileItem.previewUrl = dto.previewUrl;
                }
            }
        },
    },
    filters: {
        formatSizeFilter(size) {
            return formatSize((size))
        },
    },
    watch: {
        filePage(newValue) {
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
        searchString (searchString) {
            this.doSearch();
        },
    },
    created() {
        this.doSearch = debounce(this.doSearch, 700);
        bus.$on(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.$on(UPDATE_VIEW_FILES_DIALOG, this.updateFiles);
        bus.$on(FILE_UPLOADED, this.onFileUploaded);
    },
    destroyed() {
        bus.$off(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.$off(UPDATE_VIEW_FILES_DIALOG, this.updateFiles);
        bus.$off(FILE_UPLOADED, this.onFileUploaded);
    },
}
</script>

<style lang="stylus">
.v-card__title {
    .download-link {
        color white
    }
}
.file-info-title {
    background rgba(0, 0, 0, 0.5);
}

</style>
