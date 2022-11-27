<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ fileItemUuid ? $vuetify.lang.t('$vuetify.attached_message_files') : $vuetify.lang.t('$vuetify.attached_chat_files') }}</v-card-title>

                <v-card-text class="ma-0 pa-0">
                    <v-list v-if="!loading">
                        <template v-if="dto.count > 0">
                            <template v-for="(item, index) in dto.files">
                                <v-list-item>
                                    <v-list-item-avatar class="ma-0 pa-0">
                                        <v-btn icon v-if="canEdit(item)" @click="fireEdit(item)"><v-icon>mdi-pencil</v-icon></v-btn>
                                        <v-icon v-else>mdi-file</v-icon>
                                    </v-list-item-avatar>
                                    <v-list-item-content class="ml-4">
                                        <v-list-item-title><a :href="item.url" target="_blank">{{item.filename}}</a></v-list-item-title>
                                        <v-list-item-subtitle>
                                            {{ item.size | formatSizeFilter }}
                                            <span v-if="item.owner"> {{ $vuetify.lang.t('$vuetify.files_by') }} {{item.owner.login}}</span>
                                            <span> {{$vuetify.lang.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                            <a v-if="item.publicUrl" :href="item.publicUrl" target="_blank">
                                            {{ $vuetify.lang.t('$vuetify.files_public_url') }}
                                            </a>
                                        </v-list-item-subtitle>
                                    </v-list-item-content>


                                    <v-tooltip bottom v-if="item.canShare && !item.publicUrl">
                                        <template v-slot:activator="{ on, attrs }">
                                            <v-icon v-bind="attrs" v-on="on" class="mx-1" v-if="item.canShare && !item.publicUrl" color="primary" @click="shareFile(item, true)" dark>mdi-export</v-icon>
                                        </template>
                                        <span>{{ $vuetify.lang.t('$vuetify.share_file') }}</span>
                                    </v-tooltip>

                                    <v-tooltip bottom v-if="item.canShare && item.publicUrl">
                                        <template v-slot:activator="{ on, attrs }">
                                            <v-icon v-bind="attrs" v-on="on" class="mx-1" v-if="item.canShare && item.publicUrl" color="primary" @click="shareFile(item, false)" dark>mdi-lock</v-icon>
                                        </template>
                                        <span>{{ $vuetify.lang.t('$vuetify.unshare_file') }}</span>
                                    </v-tooltip>


                                    <v-icon class="mx-1" v-if="item.canRemove" color="error" @click="deleteFile(item)" dark>mdi-delete</v-icon>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        class="ma-4 pa-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="pa-4 d-flex flex-wrap flex-row">
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="filePage"
                        :length="filePagesCount"
                    ></v-pagination>
                    <v-spacer></v-spacer>
                    <v-btn color="primary" class="mr-4" @click="openUploadModal()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.lang.t('$vuetify.upload') }}</v-btn>
                    <v-btn color="error" class="mr-4" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    CLOSE_SIMPLE_MODAL, OPEN_FILE_UPLOAD_MODAL,
    OPEN_SIMPLE_MODAL, OPEN_TEXT_EDIT_MODAL,
    OPEN_VIEW_FILES_DIALOG, SET_FILE_ITEM_UUID, UPDATE_VIEW_FILES_DIALOG
} from "./bus";
import {mapGetters} from "vuex";
import {GET_USER} from "./store";
import axios from "axios";
import { getHumanReadableDate, replaceInArray, formatSize } from "./utils";

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
                    fileItemUuid : this.fileItemUuid ? this.fileItemUuid : ''
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
                  this.$forceUpdate();
                })
        },
        canEdit(dto) {
            return this.currentUser.id == dto.ownerId && dto.filename.endsWith('.txt');
        },
        fireEdit(dto) {
            bus.$emit(OPEN_TEXT_EDIT_MODAL, {fileInfoDto: dto, chatId: this.chatId, fileItemUuid: this.fileItemUuid});
        },
        getDate(item) {
            return getHumanReadableDate(item.lastModified)
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
        }
    },
    created() {
        bus.$on(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.$on(UPDATE_VIEW_FILES_DIALOG, this.updateFiles);
    },
    destroyed() {
        bus.$off(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.$off(UPDATE_VIEW_FILES_DIALOG, this.updateFiles);
    },
}
</script>
