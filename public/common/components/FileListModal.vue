<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" height="100%" scrollable :persistent="hasSearchString()" :fullscreen="isMobile()">
            <v-card>
                <v-card-title class="d-flex align-center ml-1 py-1">
                    <template v-if="showSearchButton">
                      Attached files
                    </template>
                    <v-spacer/>
                    <v-btn v-if="showSearchButton" :icon="fileModeIcon" variant="flat" @click="toggleFileMode" :title="fileModeTitle"></v-btn>
                    <CollapsedSearch :provider="{
                      getModelValue: this.getModelValue,
                      setModelValue: this.setModelValue,
                      getShowSearchButton: this.getShowSearchButton,
                      setShowSearchButton: this.setShowSearchButton,
                      searchName: this.searchName,
                      textFieldVariant: 'outlined',
                    }" paddings-y="true"/>

                </v-card-title>

                <v-card-text :class="isMobile() ? ['py-1', 'px-4', 'files-list'] : ['py-1', 'px-4', 'files-list']">
                    <v-row v-if="!loading">
                        <template v-if="itemsDto.count > 0">
                            <template v-if="!fileListMode">
                                <template v-for="(item, i) in itemsDto.items">
                                  <v-list-item class="list-item-prepend-spacer px-2 py-2" @contextmenu.stop="onShowContextMenu($event, item)">
                                    <template v-slot:prepend>
                                      <v-avatar v-if="hasLength(item.previewUrl)" :image="item.previewUrl"></v-avatar>
                                      <v-icon v-else class="mx-2">mdi-file</v-icon>
                                    </template>

                                    <template v-slot:default>
                                      <v-list-item-title><a :href="item.url" target="_blank" class="colored-link">{{item.filename}}</a></v-list-item-title>
                                      <v-list-item-subtitle>
                                        {{ formattedSize(item.size) }}
                                        <span v-if="item.owner"> by {{item.owner?.login}}</span>
                                        <span> at </span>{{getDate(item)}}
                                      </v-list-item-subtitle>
                                    </template>
                                  </v-list-item>
                                  <v-divider></v-divider>
                                </template>
                            </template>
                            <template v-else>
                                <v-col
                                    v-for="item in itemsDto.items"
                                    :key="item.id"
                                    :cols="isMobile() ? 12 : 6"
                                >
                                    <v-card>
                                        <v-img
                                            :src="item.previewUrl"
                                            class="align-end"
                                            cover
                                            gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
                                            height="200px"
                                        >
                                            <v-container class="file-info-title ma-0 pa-0">
                                            <v-card-title class="pb-1 card-title-wrapper">
                                              <a :href="item.url" download class="file-title download-link text-white">{{item.filename}}</a>
                                            </v-card-title>
                                            <v-card-subtitle class="text-white pb-2 no-opacity text-wrap">
                                                {{ formattedSize(item.size) }}
                                                <span v-if="item.owner"> by {{item.owner?.login}}</span>
                                                <span> at </span>{{getDate(item)}}
                                            </v-card-subtitle>
                                            </v-container>
                                        </v-img>
                                        <v-card-actions>
                                            <v-spacer></v-spacer>
                                            <a :href="item.url" download class="colored-link mx-2"><v-icon title="Download the file">mdi-download</v-icon></a>

                                            <v-btn size="medium" v-if="item.canShowAsImage" @click="fireShowImage(item)" title="View"><v-icon size="large">mdi-eye</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canPlayAsVideo" @click="fireVideoPlay(item)" title="Play"><v-icon size="large">mdi-play</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canPlayAsAudio" @click="fireAudioPlay(item)" title="Play"><v-icon size="large">mdi-play</v-icon></v-btn>

                                        </v-card-actions>
                                    </v-card>
                                </v-col>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>No files</v-card-text>
                        </template>
                    </v-row>
                    <v-progress-circular
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                    <FileListContextMenu
                        ref="contextMenuRef"
                        @showAsImage="this.fireShowImage"
                        @playAsVideo="this.fireVideoPlay"
                        @playAsAudio="this.fireAudioPlay"
                    />
                </v-card-text>

                <v-card-actions class="my-actions d-flex flex-wrap flex-row">
                  <!-- Pagination is shuddering / flickering on the second page without this wrapper -->
                  <v-row no-gutters class="ma-0 pa-0 d-flex flex-row">
                    <v-col class="ma-0 pa-0 flex-grow-1 flex-shrink-0" :class="isMobile() ? 'mb-2' : ''">
                      <v-pagination
                          variant="elevated"
                          active-color="primary"
                          :density="isMobile() ? 'compact' : 'comfortable'"
                          v-if="shouldShowPagination"
                          v-model="page"
                          :length="pagesCount"
                          :total-visible="getTotalVisible()"
                      ></v-pagination>
                      <v-divider v-if="shouldShowPagination && isMobile()" class="mt-2"/>
                    </v-col>
                    <v-col class="ma-0 pa-0 d-flex flex-row flex-grow-1 flex-shrink-0 align-self-end justify-end">
                      <v-btn color="red" variant="flat" @click="closeModal()">Close</v-btn>
                    </v-col>
                  </v-row>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
  OPEN_VIEW_FILES_DIALOG,
  PLAYER_MODAL,
} from "../bus";
import axios from "axios";
import {
    formatSize,
    hasLength,
    FIRST_PAGE, PAGE_SIZE_SMALL,
    deepCopy,
} from "../utils";
import { getHumanReadableDate } from "../date";
import debounce from "lodash/debounce";
import CollapsedSearch from "./CollapsedSearch.vue";
import Mark from "mark.js";
import {getStoredFileListMode, setStoredFileListMode} from "../localStore.js";
import FileListContextMenu from "./FileListContextMenu.vue";
import {usePageContext} from "#root/renderer/usePageContext.js";

export const dtoFactory = () => {return {items: [], count: 0} };

export default {
    setup() {
      const pageContext = usePageContext();

      // expose to template and other options API hooks
      return {
        pageContext
      }
    },

    data () {
        return {
            show: false,
            itemsDto: dtoFactory(),
            loading: false,
            page: FIRST_PAGE,
            dataLoaded: false,

            messageIdToDetachFiles: null,
            fileItemUuid: null,
            isMessageEditing: false,
            searchString: null,
            showSearchButton: true,
            markInstance: null,
            chatId: null, // overrideChatId
            messageId: null, // overrideMessageId
            fileUploadingSessionType: null,
            correlationId: null,
            fileListMode: false,
        }
    },
    computed: {
        pagesCount() {
          const count = Math.ceil(this.itemsDto.count / PAGE_SIZE_SMALL);
          return count;
        },
        shouldShowPagination() {
          return this.itemsDto != null && this.itemsDto.items && this.itemsDto.count > PAGE_SIZE_SMALL
        },

        fileModeIcon() {
          if (this.fileListMode) {
            return 'mdi-format-list-bulleted'
          } else {
            return 'mdi-image-multiple-outline'
          }
        },
        fileModeTitle() {
          if (this.fileListMode) {
            return 'Switch to list'
          } else {
            return 'Switch to miniatures'
          }
        },
    },

    methods: {
        showModal(data) {
          console.debug("Opening modal, data=", data);
          if (!this.isCachedRelevantToArguments(data)) {
            this.reset();
          }

          this.initializeWithArguments(data);

          this.show = true;

          if (!this.dataLoaded) {
            this.updateItems().then(()=>{
              if (this.onInitialized) {
                this.onInitialized()
              }
            })
          } else if (this.performMarking) {
            this.performMarking();
          }
        },
        translatePage() {
          return this.page - 1;
        },
        // smart fetching
        updateItems(silent) {
          if (!this.canUpdateItems()) {
            return Promise.resolve()
          }
          if (!silent) {
            this.loading = true;
          }
          return this.initiateRequest()
              .then((response) => {
                const dto = deepCopy(response.data);
                if (this.transformItems) {
                  this.transformItems(dto?.items);
                }
                this.itemsDto = dto;
              })
              .finally(() => {
                if (!silent) {
                  this.loading = false;
                }
                this.dataLoaded = true;
                if (this.performMarking) {
                  this.performMarking();
                }
                if (this.afterFirstDrawItems){
                  this.afterFirstDrawItems()
                }
              })
        },
        getTotalVisible() {
          if (!this.isMobile()) {
            return 7
          } else {
            if (this.page == FIRST_PAGE) {
              return 6
            } else {
              return 5
            }
          }
        },
        closeModal() {
          this.show = false;
          this.clearOnClose();
        },
        reset() {
          this.page = FIRST_PAGE;
          this.itemsDto = dtoFactory();
          this.dataLoaded = false;
          this.clearOnReset();
          this.clearOnClose();
        },

        onShowContextMenu(e, menuableItem) {
          this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
        },
        toggleFileMode() {
          const newValue = !this.fileListMode;
          this.fileListMode = newValue;
          setStoredFileListMode(newValue);
        },
        hasLength,
        isCachedRelevantToArguments({fileItemUuid, chatId, messageId}) {
            return this.fileItemUuid == fileItemUuid && this.chatId == chatId && this.messageId == messageId
        },
        initializeWithArguments({fileItemUuid, messageEditing, messageIdToDetachFiles, chatId, messageId, fileUploadingSessionType, correlationId}) {
            this.messageIdToDetachFiles = messageIdToDetachFiles;
            this.isMessageEditing = messageEditing;
            this.fileItemUuid = fileItemUuid;
            this.chatId = chatId;
            this.messageId = messageId;

            // just pass them to FileUploadModal
            this.fileUploadingSessionType = fileUploadingSessionType;
            this.correlationId = correlationId;
        },
        initiateRequest() {
            return axios.get(`/api/storage/public/${this.chatId}`, {
                params: {
                    page: this.translatePage(),
                    size: PAGE_SIZE_SMALL,
                    fileItemUuid : this.fileItemUuid,
                    searchString: this.searchString,
                    overrideChatId: this.chatId,
                    overrideMessageId: this.messageId,
                },
            })
        },
        doSearch(){
            if (!this.dataLoaded) { // we search for .mp3, then close modal, then switch to another chat
                return
            }

            this.page = FIRST_PAGE;
            this.updateItems();
        },
        transformItems(items) {
          if (items != null) {
            items.forEach(item => {
              this.transformItem(item);
            });
          }
        },
        transformItem(item) {
            item.hasNoMessage = false;
            item.loadingHasNoMessage = false;
        },
        fireVideoPlay(dto) {
            bus.emit(PLAYER_MODAL, dto);
        },
        fireAudioPlay(dto) {
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

        formattedSize(size) {
            return formatSize(size)
        },
        getModelValue() {
            return this.searchString
        },
        setModelValue(v) {
            this.searchString = v
        },
        getShowSearchButton() {
            return this.showSearchButton
        },
        setShowSearchButton(v) {
            this.showSearchButton = v
        },
        searchName() {
            return 'Search by files'
        },
        performMarking() {
          this.$nextTick(() => {
            this.markInstance.unmark();
            if (hasLength(this.searchString)) {
              this.markInstance.mark(this.searchString);
            }
          })
        },
        clearOnClose() {
            this.messageIdToDetachFiles = null;
            this.isMessageEditing = false;
            this.showSearchButton = true;
            this.fileUploadingSessionType = null;
            this.correlationId = null;
        },
        clearOnReset() {
            this.chatId = null;
            this.messageId = null;
            this.fileItemUuid = null;
            this.searchString = null;
        },
        canUpdateItems() {
          return !!this.chatId
        },
        shouldReactOnPageChange() {
            return this.show
        },

        isMobile() {
          return this.pageContext.isMobile
        },
    },
    watch: {
        show(newValue) {
          if (!newValue) {
            this.closeModal();
          }
        },
        page(newValue) {
          if (this.shouldReactOnPageChange()) {
            console.debug("Setting new page", newValue);
            this.itemsDto = dtoFactory();
            this.updateItems();
          }
        },

        searchString(searchString) {
            this.doSearch();
        },
    },
    components: {
        FileListContextMenu,
        CollapsedSearch
    },
    created() {
        this.doSearch = debounce(this.doSearch, 700);
    },
    mounted() {
      this.fileListMode = getStoredFileListMode();

      bus.on(OPEN_VIEW_FILES_DIALOG, this.showModal);
      this.markInstance = new Mark(".files-list");
    },
    beforeUnmount() {
        bus.off(OPEN_VIEW_FILES_DIALOG, this.showModal);
        this.markInstance.unmark();
        this.markInstance = null;
    },
}
</script>

<style lang="stylus" scoped>
@import "../styles/constants.styl"
.no-opacity {
  opacity 1
}
.card-title-wrapper {
  line-height 1.25em

  .file-title {
    white-space break-spaces
  }
}
.download-link {
    text-decoration none
}
.file-info-title {
    background rgba(0, 0, 0, 0.5);
}

</style>
