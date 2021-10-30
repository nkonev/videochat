<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.attached_files') }}</v-card-title>

                <v-card-text class="ma-0 pa-0">
                    <v-list v-if="!loading">
                        <template v-if="dto.files.length > 0">
                            <template v-for="(item, index) in dto.files">
                                <v-list-item>
                                    <v-list-item-avatar class="ma-0 pa-0">
                                        <v-btn icon v-if="canEdit(item)" @click="fireEdit(item)"><v-icon>mdi-pencil</v-icon></v-btn>
                                        <v-icon v-else>mdi-file</v-icon>
                                    </v-list-item-avatar>
                                    <v-list-item-content class="ml-4">
                                        <v-list-item-title><a :href="item.url" target="_blank">{{item.filename}}</a></v-list-item-title>
                                        <v-list-item-subtitle><span v-if="item.owner">by {{item.owner.login}}</span> <a v-if="item.publicUrl" :href="item.publicUrl" target="_blank">Public url</a></v-list-item-subtitle>
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

                <v-card-actions class="pa-4">
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
import {replaceInArray} from "./utils";

export default {
    data () {
        return {
            show: false,
            dto: {files: []},
            chatId: null,
            fileItemUuid: null,
            loading: false,
            messageEditing: false
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
    },

    methods: {
        showModal({chatId, fileItemUuid, messageEditing}) {
            console.log("Opening files modal, chatId=", chatId, ", fileItemUuid=", fileItemUuid);
            this.chatId = chatId;
            this.fileItemUuid = fileItemUuid;
            this.show = true;
            this.loading = true;
            this.messageEditing = messageEditing;
            this.updateFiles(()=> {
                this.loading = false;
            })
        },
        updateFiles(callback) {
            if (!this.show) {
                return
            }
            axios.get(`/api/storage/${this.chatId}` + (this.fileItemUuid ? "?fileItemUuid="+this.fileItemUuid : ""))
                .then((response) => {
                    this.dto = response.data;
                })
                .finally(() => {
                    if (callback) {
                      callback();
                    }
                })
        },
        closeModal() {
            this.show = false;
            this.chatId = null;
            this.fileItemUuid = null;
            this.messageEditing = false;
        },
        openUploadModal() {
            bus.$emit(OPEN_FILE_UPLOAD_MODAL, this.fileItemUuid, this.messageEditing, true);
        },
        deleteFile(dto) {
            bus.$emit(OPEN_SIMPLE_MODAL, {
                buttonName: 'Delete',
                title: `Delete file`,
                text: `Are you sure to delete this file '${dto.filename}' ?`,
                actionFunction: ()=> {
                    axios.delete(`/api/storage/${this.chatId}/file` + (this.fileItemUuid ? "?fileItemUuid="+this.fileItemUuid : ""), {data: {id: dto.id}})
                        .then((response) => {
                            this.dto = response.data;
                            if (this.$data.messageEditing) {
                                bus.$emit(SET_FILE_ITEM_UUID, {fileItemUuid: this.fileItemUuid, count: response.data.files.length});
                            }
                            if (this.dto.files.length == 0) {
                                this.closeModal();
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
            return dto.filename.endsWith('.txt');
        },
        fireEdit(dto) {
            bus.$emit(OPEN_TEXT_EDIT_MODAL, {fileInfoDto: dto, chatId: this.chatId, fileItemUuid: this.fileItemUuid});
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
