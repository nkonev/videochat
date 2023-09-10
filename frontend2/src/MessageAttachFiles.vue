<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.attach_files_to_message')">
                <v-card-text class="px-0">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="items.length > 0">
                            <template v-for="(item, index) in items">
                                <v-hover v-slot:default="{ hover }">
                                    <v-list-item link @click="setFileItemUuidToMessage(item)">
                                      <v-list-item-title>{{ getItemTitle(item)}}</v-list-item-title>
                                      <v-list-item-subtitle>{{ getItemSubTitle(item)}}</v-list-item-subtitle>
                                    </v-list-item>
                                </v-hover>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_chats') }}</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        class="ma-4 pa-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn
                        variant="elevated"
                        color="red"
                        @click="closeModal()"
                    >
                        {{ $vuetify.locale.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
  ATTACH_FILES_TO_MESSAGE_MODAL, SET_FILE_ITEM_FILE_COUNT, SET_FILE_ITEM_UUID,
} from "./bus/bus";
import {hasLength} from "./utils";
import axios from "axios";
import debounce from "lodash/debounce";

export default {
    data () {
        return {
            show: false,
            searchString: null,
            items: [ ], // max 20 items and search
            loading: false,
            messageId: null,
        }
    },

    methods: {
        showModal({messageId}) {
            this.show = true;
            this.messageId = messageId;
            this.loadData();
        },
        closeModal() {
            this.show = false;
            this.items = [];
            this.loading = false;
            this.searchString = null;
            this.messageId = null;
        },
        loadData() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get('/api/storage/'+this.chatId+'/file-item-uuid').then(({data}) => {
                this.items = data.files;
                this.loading = false;
            })
        },
        getItemTitle(item) {
            return item.fileItemUuid
        },
        getItemSubTitle(item) {
          return item.files.reduce((accumulator, currentValue, currentIndex) => {
            return accumulator + (currentIndex > 0 ? ", " : "") + currentValue.filename
          }, "")
        },
        setFileItemUuidToMessage(item) {
          console.log("Setting fileItemUuid to message", item)
          axios.put(`/api/chat/`+this.chatId+'/message/file-item-uuid', {
            messageId: this.messageId,
            fileItemUuid: item.fileItemUuid
          }).then(()=> {
            bus.emit(SET_FILE_ITEM_UUID, {fileItemUuid: item.fileItemUuid, chatId: this.chatId});
            bus.emit(SET_FILE_ITEM_FILE_COUNT, {count: item.files.length, chatId: this.chatId});
            this.closeModal()
          })
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
    },

    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
    },
    created() {
        bus.on(ATTACH_FILES_TO_MESSAGE_MODAL, this.showModal);
    },
    destroyed() {
        bus.off(ATTACH_FILES_TO_MESSAGE_MODAL, this.showModal);
    },
}
</script>
