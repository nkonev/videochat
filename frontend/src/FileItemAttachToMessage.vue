<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480" scrollable>
            <v-card :title="$vuetify.locale.t('$vuetify.attach_files_to_message')">
                <v-card-text class="px-0">
                    <v-list class="pb-0" v-if="!loading">
                        <template v-if="dto.files.length > 0">
                            <template v-for="(item, index) in dto.files">
                                <v-hover v-slot:default="{ hover }">
                                    <v-list-item link @click="setFileItemUuidToMessage(item)">
                                      <v-list-item-title>{{ getItemTitle(item)}}</v-list-item-title>
                                      <v-list-item-subtitle>{{ getItemSubTitle(item)}}</v-list-item-subtitle>
                                    </v-list-item>
                                </v-hover>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-list>
                    <v-progress-circular
                        class="ma-4"
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
                        :total-visible="isMobile() ? 3 : 7"
                      ></v-pagination>
                    </v-col>
                    <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-0 flex-shrink-0 align-self-end">
                      <v-btn
                        variant="elevated"
                        color="red"
                        @click="closeModal()"
                      >
                        {{ $vuetify.locale.t('$vuetify.close') }}
                      </v-btn>
                    </v-col>
                  </v-row>
                </v-card-actions>

            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
  ATTACH_FILES_TO_MESSAGE_MODAL, LOAD_FILES_COUNT, SET_FILE_ITEM_UUID,
} from "./bus/bus";
import axios from "axios";

const firstPage = 1;
const pageSize = 20;
const dtoFactory = () => {return {files: [], count: 0} };

export default {
    data () {
        return {
            show: false,
            searchString: null,
            dto: dtoFactory(),
            loading: false,
            messageId: null,
            page: firstPage,
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
            this.dto = dtoFactory();
            this.loading = false;
            this.searchString = null;
            this.messageId = null;
            this.page = firstPage;
        },
        loadData() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get('/api/storage/'+this.chatId+'/file-item-uuid', {
              params: {
                page: this.translatePage(),
                size: pageSize,
              },
            }).then(({data}) => {
                this.dto = data;
                this.loading = false;
            })
        },
        translatePage() {
            return this.page - 1;
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
            bus.emit(LOAD_FILES_COUNT, {chatId: this.chatId});
            this.closeModal()
          })
        },
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        pagesCount() {
            const count = Math.ceil(this.dto.count / pageSize);
            // console.debug("Calc pages count", count);
            return count;
        },
        shouldShowPagination() {
            return this.dto != null && this.dto.files && this.dto.count > pageSize
        },
    },

    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        },
        page(newValue) {
          if (this.show) {
            console.debug("SettingNewPage", newValue);
            this.dto = dtoFactory();
            this.loadData();
          }
        },
    },
    mounted() {
        bus.on(ATTACH_FILES_TO_MESSAGE_MODAL, this.showModal);
    },
    beforeUnmount() {
        bus.off(ATTACH_FILES_TO_MESSAGE_MODAL, this.showModal);
    },
}
</script>
