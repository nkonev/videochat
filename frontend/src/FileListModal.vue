<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="800" height="100%" scrollable :persistent="hasSearchString()" :fullscreen="isMobile()">
            <v-card>
                <v-card-title class="d-flex align-center ml-1 py-1">
                    <template v-if="showSearchButton">
                        {{ fileItemUuid ? $vuetify.locale.t('$vuetify.attached_message_files') : $vuetify.locale.t('$vuetify.attached_chat_files') }}
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
                                        <span v-if="item.owner"> {{ $vuetify.locale.t('$vuetify.files_by') }} {{item.owner?.login}}</span>
                                        <span> {{$vuetify.locale.t('$vuetify.time_at')}} </span>{{getDate(item)}}
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
                                                <span v-if="item.owner"> {{ $vuetify.locale.t('$vuetify.files_by') }} {{item.owner?.login}}</span>
                                                <span> {{$vuetify.locale.t('$vuetify.time_at')}} </span>{{getDate(item)}}
                                                <a v-if="item.publishedUrl" :href="item.publishedUrl" target="_blank" class="colored-link">
                                                    {{ $vuetify.locale.t('$vuetify.files_published_url') }}
                                                </a>
                                            </v-card-subtitle>
                                            </v-container>
                                        </v-img>
                                        <v-card-actions>
                                            <v-spacer></v-spacer>
                                            <a :href="item.url" download class="colored-link mx-2"><v-icon :title="$vuetify.locale.t('$vuetify.file_download')">mdi-download</v-icon></a>

                                            <v-btn size="medium" :disabled="item.hasNoMessage" :loading="item.loadingHasNoMessage" @click="fireSearchMessage(item)" :title="$vuetify.locale.t('$vuetify.search_related_message')"><v-icon size="large">mdi-note-search-outline</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canShowAsImage" @click="fireShowImage(item)" :title="$vuetify.locale.t('$vuetify.view')"><v-icon size="large">mdi-eye</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canPlayAsVideo" @click="fireVideoPlay(item)" :title="$vuetify.locale.t('$vuetify.play')"><v-icon size="large">mdi-play</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canPlayAsAudio" @click="fireAudioPlay(item)" :title="$vuetify.locale.t('$vuetify.play')"><v-icon size="large">mdi-play</v-icon></v-btn>

                                            <v-btn size="medium" v-if="item.canEdit" @click="fireEdit(item)" :title="$vuetify.locale.t('$vuetify.edit')"><v-icon size="large">mdi-pencil</v-icon></v-btn>

                                            <template v-if="item.canShare">
                                                <v-btn size="medium" v-if="!item.publishedUrl" @click="shareFile(item)">
                                                    <v-icon color="primary" size="large" dark :title="$vuetify.locale.t('$vuetify.share_file')">mdi-export</v-icon>
                                                </v-btn>

                                                <v-btn size="medium" v-if="item.publishedUrl" @click="unshareFile(item)">
                                                  <v-icon color="primary" size="large" dark :title="$vuetify.locale.t('$vuetify.unshare_file')">mdi-lock</v-icon>
                                                </v-btn>
                                            </template>
                                            <v-btn size="medium" v-if="item.canDelete" @click="deleteFile(item)">
                                                <v-icon color="red" size="large" dark :title="$vuetify.locale.t('$vuetify.delete_btn')">mdi-delete</v-icon>
                                            </v-btn>
                                        </v-card-actions>
                                    </v-card>
                                </v-col>
                            </template>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.locale.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-row>
                    <v-progress-circular
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                    <FileListContextMenu
                        ref="contextMenuRef"
                        @searchRelatedMessage="this.fireSearchMessage"
                        @showAsImage="this.fireShowImage"
                        @playAsVideo="this.fireVideoPlay"
                        @playAsAudio="this.fireAudioPlay"
                        @edit="this.fireEdit"
                        @share="this.shareFile"
                        @unshare="this.unshareFile"
                        @delete="this.deleteFile"
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
  MESSAGE_EDIT_SET_FILE_ITEM_UUID,
  FILE_CREATED,
  FILE_REMOVED,
  PLAYER_MODAL,
  FILE_UPDATED, MESSAGE_EDIT_LOAD_FILES_COUNT, LOGGED_OUT, REFRESH_ON_WEBSOCKET_RESTORED
} from "./bus/bus";
import axios from "axios";
import {
    formatSize, getUrlPrefix,
    hasLength,
} from "./utils";
import { getHumanReadableDate } from "@/date.js";
import debounce from "lodash/debounce";
import CollapsedSearch from "@/CollapsedSearch.vue";
import Mark from "mark.js";
import {messageIdHashPrefix} from "@/router/routes";
import pageableModalMixin, {firstPage, pageSize} from "@/mixins/pageableModalMixin.js";
import {getStoredFileListMode, setStoredFileListMode} from "@/store/localStore.js";
import FileListContextMenu from "@/FileListContextMenu.vue";

export default {
    mixins: [pageableModalMixin()],
    data () {
        return {
            messageIdToDetachFiles: null,
            fileItemUuid: null,
            isOpenedFromMessageEditing: false, // is opened from message editing
            searchString: null,
            showSearchButton: true,
            markInstance: null,
            chatId: null,
            fileUploadingSessionType: null,
            correlationId: null,
            fileListMode: false,
        }
    },
    computed: {
        fileModeIcon() {
          if (this.fileListMode) {
            return 'mdi-format-list-bulleted'
          } else {
            return 'mdi-image-multiple-outline'
          }
        },
        fileModeTitle() {
          if (this.fileListMode) {
            return this.$vuetify.locale.t('$vuetify.file_mode_switch_to_list')
          } else {
            return this.$vuetify.locale.t('$vuetify.file_mode_switch_to_miniatures')
          }
        },
    },

    methods: {
        onShowContextMenu(e, menuableItem) {
          this.$refs.contextMenuRef.onShowContextMenu(e, menuableItem);
        },
        toggleFileMode() {
          const newValue = !this.fileListMode;
          this.fileListMode = newValue;
          setStoredFileListMode(newValue);
        },
        hasLength,
        isCachedRelevantToArguments({fileItemUuid, chatId}) {
            return this.fileItemUuid == fileItemUuid && this.chatId == chatId
        },
        initializeWithArguments({fileItemUuid, messageEditing, messageIdToDetachFiles, chatId, fileUploadingSessionType, correlationId}) {
            this.messageIdToDetachFiles = messageIdToDetachFiles;
            this.isOpenedFromMessageEditing = messageEditing;
            this.fileItemUuid = fileItemUuid;
            this.chatId = chatId;

            // just pass them to FileUploadModal
            this.fileUploadingSessionType = fileUploadingSessionType;
            this.correlationId = correlationId;
        },
        initiateRequest() {
            return axios.get(`/api/storage/${this.chatId}`, {
                params: {
                    page: this.translatePage(),
                    size: pageSize,
                    fileItemUuid : this.fileItemUuid ? this.fileItemUuid : '',
                    searchString: this.searchString
                },
            })
        },
        doSearch(){
            if (!this.dataLoaded) { // we search for .mp3, then close modal, then switch to another chat
                return
            }

            this.page = firstPage;
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
        openUploadModal() {
            bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: this.fileItemUuid, shouldSetFileUuidToMessage: this.isOpenedFromMessageEditing, fileUploadingSessionType: this.fileUploadingSessionType, correlationId: this.correlationId});
        },
        onDetachFilesFromMessage () {
          axios.put(`/api/chat/`+this.chatId+'/message/file-item-uuid', {
            messageId: this.messageIdToDetachFiles,
            fileItemUuid: null
          }).then(()=>{
            bus.emit(MESSAGE_EDIT_SET_FILE_ITEM_UUID, {fileItemUuid: null, chatId: this.chatId});
            bus.emit(MESSAGE_EDIT_LOAD_FILES_COUNT, {chatId: this.chatId});
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
                        bus.emit(CLOSE_SIMPLE_MODAL);
                    }).finally(()=>{
                      that.loading = false;
                    })
                }
            });
        },
        shareFile(dto) {
            axios.put(`/api/storage/publish/file`, {id: dto.id, public: true}).then((resp)=>{
                const link = resp.data.publishedUrl;
                if (link) {
                    navigator.clipboard.writeText(getUrlPrefix() + link);
                    this.setTempNotification(this.$vuetify.locale.t('$vuetify.published_file_link_copied'));
                }
            })
        },
        unshareFile(dto) {
            axios.put(`/api/storage/publish/file`, {id: dto.id, public: false})
        },
        fireEdit(dto) {
            bus.emit(OPEN_TEXT_EDIT_MODAL, {fileInfoDto: dto, chatId: this.chatId, fileItemUuid: this.fileItemUuid});
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
        fireSearchMessage(dto) {
            dto.loadingHasNoMessage = true
            axios.get("/api/chat/"+this.chatId+"/message/find-by-file-item-uuid/" + dto.fileItemUuid)
              .then(response => {
                if (response.status == 204) {
                  dto.hasNoMessage = true
                } else {
                  const name = this.$route.name;
                  this.$router.push({
                    name: name,
                    params: {
                      id: this.chatId
                    },
                    hash: messageIdHashPrefix + response.data.messageId,
                  }).then(()=>{
                    if (this.isMobile()) {
                      this.closeModal()
                    }
                  })
                }
              }).finally(()=>{
                dto.loadingHasNoMessage = false
              })
        },
        getDate(item) {
            return getHumanReadableDate(item.lastModified)
        },
        hasSearchString() {
            return hasLength(this.searchString)
        },

        onPreviewCreated(dto) {
          if (!this.dataLoaded) {
            return
          }
          console.log("Replacing preview", dto);
          for (const fileItem of this.itemsDto.items) {
            if (fileItem.id == dto.id) {
              fileItem.previewUrl = dto.previewUrl;
              break
            }
          }
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
            return this.$vuetify.locale.t('$vuetify.search_by_files')
        },
        performMarking() {
          this.$nextTick(() => {
            this.markInstance.unmark();
            if (hasLength(this.searchString)) {
              this.markInstance.mark(this.searchString);
            }
          })
        },
        extractDtoFromEventDto(eventDto) {
            return [eventDto.fileInfoDto]
        },
        initiateFilteredRequest(eventDto) {
            return axios.post(`/api/storage/${this.chatId}/file/filter`, {
                fileItemUuid: this.fileItemUuid,
                searchString: this.searchString,
                fileId: eventDto.fileInfoDto.id
            })
        },
        initiateCountRequest() {
            return axios.post(`/api/storage/${this.chatId}/file/count`, {
                fileItemUuid: this.fileItemUuid,
                searchString: this.searchString,
            })
        },
        clearOnClose() {
            this.messageIdToDetachFiles = null;
            this.isOpenedFromMessageEditing = false;
            this.showSearchButton = true;
            this.fileUploadingSessionType = null;
            this.correlationId = null;
        },
        clearOnReset() {
            this.chatId = null;
            this.fileItemUuid = null;
            this.searchString = null;
        },
        canUpdateItems() {
          return !!this.chatId
        },
        resetOnRouteIdChange(){
            return true
        },
        shouldReactOnPageChange() {
            return this.show
        },
        debouncedUpdate() {
          this.updateItems();
        },
        onWsRestoredRefresh() {
          if (this.dataLoaded) {
            this.debouncedUpdate();
          }
        },
    },
    watch: {
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
        this.debouncedUpdate = debounce(this.debouncedUpdate, 300, {leading:false, trailing:true})
    },
    mounted() {
      this.fileListMode = getStoredFileListMode();

      bus.on(OPEN_VIEW_FILES_DIALOG, this.showModal);
      bus.on(PREVIEW_CREATED, this.onPreviewCreated);
      bus.on(FILE_CREATED, this.onItemCreatedEvent);
      bus.on(FILE_UPDATED, this.onItemUpdatedEvent);
      bus.on(FILE_REMOVED, this.onItemRemovedEvent);
      bus.on(LOGGED_OUT, this.onLogout);
      bus.on(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
      this.markInstance = new Mark(".files-list");
    },
    beforeUnmount() {
        bus.off(OPEN_VIEW_FILES_DIALOG, this.showModal);
        bus.off(PREVIEW_CREATED, this.onPreviewCreated);
        bus.off(FILE_CREATED, this.onItemCreatedEvent);
        bus.off(FILE_UPDATED, this.onItemUpdatedEvent);
        bus.off(FILE_REMOVED, this.onItemRemovedEvent);
        bus.off(LOGGED_OUT, this.onLogout);
        bus.off(REFRESH_ON_WEBSOCKET_RESTORED, this.onWsRestoredRefresh);
        this.markInstance.unmark();
        this.markInstance = null;
    },
}
</script>

<style lang="stylus" scoped>
@import "constants.styl"
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
