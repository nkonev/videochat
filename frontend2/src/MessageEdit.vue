<template>
    <v-container class="ma-0 pa-0" style="height: 100%" fluid>
      <v-container id="sendButtonContainer" class="py-0 px-0 pr-1 pb-1 d-flex flex-column" fluid>
            <div class="answer-wrapper" v-if="showAnswer">
              <div class="answer-text"><v-icon @click="resetAnswer()" :title="$vuetify.locale.t('$vuetify.remove_answer')">mdi-close</v-icon>{{answerOnPreview}}</div>
            </div>
              <tiptap
                  ref="tipTapRef"
                  @input="onInput"
                  @keydown.ctrl.enter.native="sendMessageToChat"
                  @keydown.esc.native="resetInput()"
              />

                  <div class="d-flex flex-wrap flex-row dashed-borders">
                      <div style="max-width: 100%" v-if="this.$refs.tipTapRef">
                          <v-slide-group
                              multiple
                              show-arrows
                          >
                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="boldValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="boldValue()" @click="boldClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_bold')">
                                  <v-icon :size="getIconSize()">mdi-format-bold</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="italicValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="italicValue()" @click="italicClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_italic')">
                                  <v-icon :size="getIconSize()">mdi-format-italic</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="underlineValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="underlineValue()" @click="underlineClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_underline')">
                                  <v-icon :size="getIconSize()">mdi-format-underline</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="strikeValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="strikeValue()" @click="strikeClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_strike')">
                                  <v-icon :size="getIconSize()">mdi-format-strikethrough-variant</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="codeValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="codeValue()" @click="codeClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_code')">
                                  <v-icon :size="getIconSize()">mdi-code-braces</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" :variant="linkValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="linkValue()" :disabled="linkButtonDisabled()" @click="linkClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_link')">
                                  <v-icon :size="getIconSize()">mdi-link-variant</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="imageClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_image')">
                                  <v-icon :size="getIconSize()">mdi-image-outline</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="videoClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_video')">
                                  <v-icon :size="getIconSize()">mdi-video</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="embedClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_embed')">
                                  <v-icon :size="getIconSize()">mdi-youtube</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="textColorClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_text_color')">
                                  <v-icon :size="getIconSize()">mdi-invert-colors</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="backgroundColorClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_background_color')">
                                  <v-icon :size="getIconSize()">mdi-format-color-fill</v-icon>
                              </v-btn>

                              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="smileyClick" :width="getBtnWidth()" :title="$vuetify.locale.t('$vuetify.message_edit_smiley')">
                                  <v-icon :size="getIconSize()">mdi-emoticon-outline</v-icon>
                              </v-btn>

                          </v-slide-group>
                      </div>

                      <div class="custom-toolbar-send">
                          <v-btn v-if="!this.editMessageDto.fileItemUuid" icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" @click="openFileUploadForAddingFiles()" :title="$vuetify.locale.t('$vuetify.message_edit_file')">
                            <v-icon :size="getIconSize()" color="primary">mdi-file-upload</v-icon>
                          </v-btn>
                          <template v-if="this.editMessageDto.fileItemUuid">
                              <v-badge
                                  :value="fileCount"
                                  :content="fileCount"
                                  color="green"
                                  overlap
                                  left
                              >
                                  <v-btn icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" @click="onFilesClicked()" :title="$vuetify.locale.t('$vuetify.message_edit_attached_files')">
                                    <v-icon :size="getIconSize()">mdi-file-document-multiple</v-icon>
                                  </v-btn>
                              </v-badge>
                          </template>
                          <v-btn icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" class="mr-2" @click="resetInput()" :title="$vuetify.locale.t('$vuetify.message_edit_clear')">
                            <v-icon :size="getIconSize()">mdi-delete</v-icon>
                          </v-btn>
                          <v-switch v-if="chatStore.canBroadcast" dense hide-details class="ma-0 mr-4" v-model="sendBroadcast"
                              :label="$vuetify.locale.t('$vuetify.message_broadcast')"
                          ></v-switch>
                          <v-btn color="primary" @click="sendMessageToChat" rounded="0" class="mr-0 send" density="comfortable" icon="mdi-send" width="64" :title="$vuetify.locale.t('$vuetify.message_edit_send')" :disabled="sending" :loading="sending"></v-btn>
                      </div>

                  </div>
      </v-container>

      <!-- We store modals outside of container in order they not to contribute into the height (as it is done in App.vue) -->
      <MessageEditLinkModal/>
      <MessageEditColorModal/>
      <MessageEditMediaModal/>
      <MessageEditSmileyModal/>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
      CLOSE_EDIT_MESSAGE, INCREMENT_FILE_ITEM_FILE_COUNT, LOAD_FILES_COUNT,
      MESSAGE_EDIT_COLOR_SET,
      MESSAGE_EDIT_LINK_SET,
      OPEN_FILE_UPLOAD_MODAL,
      OPEN_MESSAGE_EDIT_COLOR,
      OPEN_MESSAGE_EDIT_LINK,
      OPEN_MESSAGE_EDIT_MEDIA,
      OPEN_MESSAGE_EDIT_SMILEY,
      OPEN_VIEW_FILES_DIALOG,
      PROFILE_SET,
      SET_EDIT_MESSAGE,
      SET_FILE_ITEM_UUID,
    } from "./bus/bus";
    import debounce from "lodash/debounce";
    import Tiptap from './TipTapEditor.vue'
    import {
        chatEditMessageDtoFactory,
        colorBackground,
        colorText, embed, getAnswerPreviewFields, link_dialog_type_add_link_to_text, link_dialog_type_add_media_embed,
        media_image, media_video
    } from "@/utils";
    import {
        getStoredChatEditMessageDto,
        removeStoredChatEditMessageDto,
        setStoredChatEditMessageDto
    } from "@/store/localStore"

    import MessageEditLinkModal from "@/MessageEditLinkModal";
    import MessageEditColorModal from "@/MessageEditColorModal";
    import MessageEditMediaModal from "@/MessageEditMediaModal";
    import MessageEditSmileyModal from "@/MessageEditSmileyModal.vue";
    import {mapStores} from "pinia";
    import {useChatStore} from "@/store/chatStore";

    export default {
        props:['chatId'],
        data() {
            return {
                editMessageDto: chatEditMessageDtoFactory(),
                fileCount: null,
                sendBroadcast: false,
                sending: false,
                showAnswer: false,
                answerOnPreview: null,
            }
        },
        methods: {
            getContent(){
                return this.$refs.tipTapRef.getContent();
            },
            sendMessageToChat() {
                if (this.messageTextIsPresent(this.editMessageDto.text)) {
                    this.sending = true;
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto))
                        .then(response => {
                            this.resetInput();
                            bus.emit(CLOSE_EDIT_MESSAGE);
                        }).finally(() => {
                            this.sending = false;
                        })
                }
            },
            resetInput() {
              this.removeFromStore();

              console.log("Resetting text input");
              this.$refs.tipTapRef.clearContent();
              this.editMessageDto = chatEditMessageDtoFactory();
              this.resetAnswer();
              this.fileCount = null;
              this.notifyAboutBroadcast(true);
            },
            messageTextIsPresent(text) {
                return text && text !== ""
            },
            loadFilesCount() {
                if (this.editMessageDto.fileItemUuid) {
                    return axios.get(`/api/storage/${this.chatId}/file/count/${this.editMessageDto.fileItemUuid}`)
                        .then((response) => {
                            this.fileCount = response.data.count;
                            return response
                        });
                } else {
                  return Promise.resolve(false)
                }
            },
            loadFilesCountAndResetFileItemUuidIfNeed() {
              this.loadFilesCount().then((resp) => {
                if (resp) {
                  if (this.fileCount === 0) {
                    this.editMessageDto.fileItemUuid = null;
                    this.saveToStore();
                  }
                }
              })
            },

            resetAnswer() {
                this.showAnswer = false;
                this.answerOnPreview = null;
                this.editMessageDto.embedMessage = null;
                this.saveToStore();
            },
            loadEmbedPreviewIfNeed(dto) {
                if (dto.embedMessage?.id) {
                    const {embedPreviewText, embedPreviewOwner} = getAnswerPreviewFields(dto);
                    axios.put('/api/chat/public/preview-without-html', {text: embedPreviewText, login: embedPreviewOwner}).then(({data}) => {
                        this.showAnswer = true;
                        this.answerOnPreview = data.text;
                    })
                }
            },
            onSetMessage(dto) {
                if (!dto) {
                    // opening modal from mobile, just from scratch
                    this.loadFromStore();
                } else {
                    this.loadEmbedPreviewIfNeed(dto);
                    this.editMessageDto = dto;
                    this.saveToStore();
                    this.$refs.tipTapRef.setContent(this.editMessageDto.text);
                    this.loadFilesCount();
                }
                this.$nextTick(()=>{
                    this.$refs.tipTapRef.setCursorToEnd()
                })
            },
            notifyAboutBroadcast(clear, val) {
                if (clear) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: null});
                } else if (this.messageTextIsPresent(val)) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: val});
                }
            },
            notifyAboutTyping(val) {
                if (this.messageTextIsPresent(val)) {
                    axios.put(`/api/chat/` + this.chatId + '/typing');
                }
            },
            onInput(val) {
                this.editMessageDto.text = this.getContent();

                this.saveToStore();

                if (this.sendBroadcast) {
                    this.notifyAboutBroadcast(false, val);
                } else {
                    this.notifyAboutTyping(val);
                }
            },
            getBtnWidth() {
                if (this.isMobile()) {
                    return '64px'
                } else {
                    return '48px'
                }
            },
            getBtnSize() {
              if (this.isMobile()) {
                return 'large'
              } else {
                return undefined
              }
            },
            getIconSize() {
              if (this.isMobile()) {
                return 'large'
              } else {
                return undefined
              }
            },
            openFileUploadForAddingFiles() {
                bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: this.editMessageDto.fileItemUuid, shouldSetFileUuidToMessage: true, messageIdToAttachFiles: this.editMessageDto.id});
            },
            onFilesClicked() {
                bus.emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid: this.editMessageDto.fileItemUuid, messageEditing: true});
            },
            onIncrementFileItemFileCount({chatId}) {
                if (this.chatId == chatId) {
                    this.fileCount++
                }
            },
            onFileItemUuid({fileItemUuid, chatId}) {
              if (chatId == this.chatId) {
                this.editMessageDto.fileItemUuid = fileItemUuid;
                this.saveToStore();
              }
            },
            boldValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('bold')
            },
            boldClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleBold().run()
            },
            italicValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('italic')
            },
            italicClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleItalic().run()
            },
            underlineValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('underline')
            },
            underlineClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleUnderline().run()
            },
            strikeValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('strike')
            },
            codeValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('code')
            },
            strikeClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleStrike().run()
            },
            codeClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleCode().run()
            },
            linkClick() {
                const previousUrl = this.$refs.tipTapRef.$data.editor.getAttributes('link').href;
                bus.emit(OPEN_MESSAGE_EDIT_LINK, {dialogType: link_dialog_type_add_link_to_text, previousUrl});
            },
            onMessageLinkSet(url) {
                // empty
                if (url === '') {
                    this.$refs.tipTapRef.$data.editor
                        .chain()
                        .focus()
                        .extendMarkRange('link')
                        .unsetLink()
                        .run()
                    return
                }

                // update link
                this.$refs.tipTapRef.$data.editor
                    .chain()
                    .focus()
                    .extendMarkRange('link')
                    .setLink({ href: url })
                    .run()
            },
            linkValue() {
                return this.$refs.tipTapRef.$data.editor.isActive('link')
            },
            linkButtonDisabled() {
                return this.$refs.tipTapRef.$data.editor.view.state.selection.empty;
            },
            imageClick() {
                bus.emit(
                  OPEN_MESSAGE_EDIT_MEDIA,
                  {
                    type: media_image,
                    fromDiskCallback: () => this.$refs.tipTapRef.addImage(),
                    setExistingMediaCallback: this.$refs.tipTapRef.setImage
                  }
                );
            },
            videoClick() {
                bus.emit(
                  OPEN_MESSAGE_EDIT_MEDIA,
                  {
                    type: media_video,
                    fromDiskCallback: () => this.$refs.tipTapRef.addVideo(),
                    setExistingMediaCallback: this.$refs.tipTapRef.setVideo
                  },
                );
            },
            embedClick() {
                bus.emit(OPEN_MESSAGE_EDIT_LINK, {dialogType: link_dialog_type_add_media_embed, mediaType: embed});
            },
            textColorClick(){
                bus.emit(OPEN_MESSAGE_EDIT_COLOR, colorText);
            },
            backgroundColorClick() {
                bus.emit(OPEN_MESSAGE_EDIT_COLOR, colorBackground);
            },
            smileyClick() {
                bus.emit(OPEN_MESSAGE_EDIT_SMILEY, (text) => this.$refs.tipTapRef.addText(text));
            },
            onColorSet({color, colorMode}) {
                console.debug("Setting color", color, colorMode);
                if (colorMode == colorText) {
                    if (color) {
                        this.$refs.tipTapRef.$data.editor.chain().focus().setColor(color).run()
                    } else {
                        this.$refs.tipTapRef.$data.editor.chain().focus().unsetColor().run()
                    }
                } else if (colorMode == colorBackground) {
                    if (color) {
                        this.$refs.tipTapRef.$data.editor.chain().focus().setHighlight({ color: color }).run()
                    } else {
                        this.$refs.tipTapRef.$data.editor.chain().focus().unsetHighlight().run()
                    }
                }
            },
            onProfileSet(){
                this.loadFromStore();
            },
            loadFromStore() {
                this.editMessageDto = getStoredChatEditMessageDto(this.chatId, chatEditMessageDtoFactory());
                if (this.editMessageDto.ownerId && this.editMessageDto.ownerId != this.chatStore.currentUser?.id) {
                    console.log("Removing owner from saved message")
                    this.editMessageDto.ownerId = null;
                    this.editMessageDto.id = null;
                }
                this.loadEmbedPreviewIfNeed(this.editMessageDto);
                this.$nextTick(()=>{
                    this.$refs.tipTapRef.setContent(this.editMessageDto.text);
                });
                this.loadFilesCount();
            },
            saveToStore() {
                setStoredChatEditMessageDto(this.editMessageDto, this.chatId);
            },
            removeFromStore() {
                removeStoredChatEditMessageDto(this.chatId);
            },
        },
        computed: {
          ...mapStores(useChatStore),
            userIsSet() {
                return !!this.chatStore.currentUser
            }
        },
        mounted() {
            bus.on(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.on(SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.on(INCREMENT_FILE_ITEM_FILE_COUNT, this.onIncrementFileItemFileCount);
            bus.on(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.on(MESSAGE_EDIT_COLOR_SET, this.onColorSet);
            bus.on(PROFILE_SET, this.onProfileSet);
            bus.on(LOAD_FILES_COUNT, this.loadFilesCountAndResetFileItemUuidIfNeed);
            this.loadFromStore();
        },
        beforeDestroy() {
            bus.off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.off(SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.off(INCREMENT_FILE_ITEM_FILE_COUNT, this.onIncrementFileItemFileCount);
            bus.off(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.off(MESSAGE_EDIT_COLOR_SET, this.onColorSet);
            bus.off(PROFILE_SET, this.onProfileSet);
            bus.off(LOAD_FILES_COUNT, this.loadFilesCountAndResetFileItemUuidIfNeed);
        },
        created(){
            this.notifyAboutTyping = debounce(this.notifyAboutTyping, 500, {leading:true, trailing:false});
            this.notifyAboutBroadcast = debounce(this.notifyAboutBroadcast, 100, {leading:true, trailing:true});
        },
        watch: {
            sendBroadcast: {
                handler: function (newValue, oldValue) {
                    if (!newValue) {
                        this.notifyAboutBroadcast(true);
                    } else {
                        this.notifyAboutBroadcast(false, this.getContent());
                    }
                }
            },
            '$vuetify.locale.current': {
                handler: function (newValue, oldValue) {
                    // reload
                    this.$refs.tipTapRef.clearContent();
                    this.$nextTick(() => {
                        this.loadFromStore();
                    })
                },
            },
        },
        components: {
            Tiptap,
            MessageEditLinkModal,
            MessageEditColorModal,
            MessageEditMediaModal,
            MessageEditSmileyModal,
        }
    }
</script>

<style lang="stylus">
@import "common.styl"

#sendButtonContainer {
    background white
    min-height 25%
    height 100%
}

.richText {
    border-color: $borderColor
    line-height: $lineHeight;
}

.dashed-borders {
    border-top-width: 0
    border-bottom-style dashed
    border-left-style dashed
    border-right-style dashed
    border-width 1px
    border-color: $borderColor
}

@media screen and (max-width: $mobileWidth) {
    .dashed-borders {
        border-width: 0
    }
}

.custom-toolbar-send {
    display: flex;
    flex-grow: 10
    justify-content flex-end
    align-items center
}

.answer-wrapper {
    display: block
}
.answer-text {
    background: $embedMessageColor;
    white-space: nowrap;
    text-overflow: ellipsis;
    overflow: hidden;
}
</style>
