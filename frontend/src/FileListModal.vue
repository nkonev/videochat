<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" persistent>
            <v-card>
                <v-card-title>Attached files</v-card-title>

                <v-container fluid>
                    <v-list v-if="dto.files.length > 0">
                        <template v-for="(item, index) in dto.files">
                            <v-list-item class="pl-0 ml-1 pr-0 mr-1 mb-1 mt-1">
                                <v-list-item-avatar class="ma-0 pa-0">
                                    <v-icon>mdi-file</v-icon>
                                </v-list-item-avatar>
                                <v-list-item-content class="ml-4">
                                    <v-list-item-title>{{item.filename}}</v-list-item-title>
                                </v-list-item-content>
                            </v-list-item>
                            <v-divider></v-divider>
                        </template>
                    </v-list>
                    <v-progress-circular
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>

                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="error" class="mr-4" @click="closeModal()">Close</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_VIEW_FILES_DIALOG
} from "./bus";
import {mapGetters} from "vuex";
import {GET_USER} from "./store";
import axios from "axios";

export default {
    data () {
        return {
            show: false,
            dto: {files: [{name: 'a.png'}, {name: 'Файл.txt'}]},
            chatId: null,
        }
    },
    computed: {
        ...mapGetters({currentUser: GET_USER}), // currentUser is here, 'getUser' -- in store.js
    },

    methods: {
        showModal(chatId) {
            this.chatId = chatId;
            this.show = true;
            axios.get(`/api/storage/${this.chatId}`)
                .then((response) => {
                    this.dto = response.data;
                })
        },
        closeModal() {
            this.show = false;
            this.chatId = null;
        },

    },

    created() {
        bus.$on(OPEN_VIEW_FILES_DIALOG, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_VIEW_FILES_DIALOG, this.showModal);
    },
}
</script>
