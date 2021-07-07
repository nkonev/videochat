<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent scrollable>
            <v-card>
                <v-card-title>Attached files</v-card-title>

                <v-card-text class="ma-0 pa-0">
                    <v-list v-if="!loading">
                        <template v-if="dto.files.length > 0">
                            <template v-for="(item, index) in dto.files">
                                <v-list-item>
                                    <v-list-item-avatar class="ma-0 pa-0">
                                        <v-icon>mdi-file</v-icon>
                                    </v-list-item-avatar>
                                    <v-list-item-content class="ml-4">
                                        <v-list-item-title><a :href="item.url" target="_blank">{{item.filename}}</a></v-list-item-title>
                                        <v-list-item-subtitle v-if="item.owner">by {{item.owner.login}}</v-list-item-subtitle>
                                    </v-list-item-content>
                                    <v-icon class="mx-1" v-if="item.canRemove" color="error" @click="deleteFile(item)" dark>mdi-delete</v-icon>
                                </v-list-item>
                                <v-divider></v-divider>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>No files</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="closeModal()">Close</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    CLOSE_SIMPLE_MODAL,
    OPEN_SIMPLE_MODAL,
    OPEN_VIEW_FILES_DIALOG
} from "./bus";
import {mapGetters} from "vuex";
import {GET_USER} from "./store";
import axios from "axios";

export default {
    data () {
        return {
            show: false,
            dto: {files: []},
            chatId: null,
            fileItemUuid: null,
            loading: false,
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
    },

    methods: {
        showModal({chatId, fileItemUuid}) {
            console.log("Opening files modal, chatId=", chatId, ", fileItemUuid=", fileItemUuid);
            this.chatId = chatId;
            this.fileItemUuid = fileItemUuid;
            this.show = true;
            this.loading = true;
            axios.get(`/api/storage/${this.chatId}` + (this.fileItemUuid ? "?fileItemUuid="+this.fileItemUuid : ""))
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
        },
        deleteFile(dto) {
            bus.$emit(OPEN_SIMPLE_MODAL, {
                buttonName: 'Delete',
                title: `Delete file`,
                text: `Are you sure to delete this file '${dto.filename}' ?`,
                actionFunction: ()=> {
                    axios.delete(`/api/storage/${this.chatId}/file`, {data: {id: dto.id}})
                        .then((response) => {
                            this.dto = response.data
                            if (this.dto.files.length == 0) {
                                this.closeModal();
                            }
                            bus.$emit(CLOSE_SIMPLE_MODAL);
                        })
                }
            });
        }
    },

    created() {
        bus.$on(OPEN_VIEW_FILES_DIALOG, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_VIEW_FILES_DIALOG, this.showModal);
    },
}
</script>
