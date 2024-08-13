<template>
    <v-container class="ma-0 pa-0" style="height: 100%" fluid>
      <v-container id="sendButtonContainer" class="py-0 px-0 d-flex flex-column" fluid>
        <div class="answer-wrapper" v-if="showAnswer">
          <div class="answer-text"><v-icon @click="resetAnswer()" :title="$vuetify.locale.t('$vuetify.remove_answer')">mdi-close</v-icon><span v-html="answerOnPreview"></span></div>
        </div>
        <tiptap
            ref="tipTapRef"
            @myinput="onInput"
            @keydown.ctrl.enter.native="sendMessageToChat"
            @keydown.esc.native="resetInput()"
            @sendMessage="sendMessageToChat"
        />

        <div class="d-flex flex-wrap flex-row align-center" v-if="chatStore.shouldShowSendMessageButtons">
          <v-slide-group
              multiple
              show-arrows
          >
              <v-btn icon rounded="0" :size="getBtnSize()" :variant="boldValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="boldValue()" @click="boldClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_bold')">
                  <v-icon :size="getIconSize()">mdi-format-bold</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="italicValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="italicValue()" @click="italicClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_italic')">
                  <v-icon :size="getIconSize()">mdi-format-italic</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="underlineValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="underlineValue()" @click="underlineClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_underline')">
                  <v-icon :size="getIconSize()">mdi-format-underline</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="strikeValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="strikeValue()" @click="strikeClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_strike')">
                  <v-icon :size="getIconSize()">mdi-format-strikethrough-variant</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="codeValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="codeValue()" @click="codeClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_code')">
                  <v-icon :size="getIconSize()">mdi-code-braces</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="codeBlockValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="codeBlockValue()" @click="codeBlockClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_code_block')">
                  <v-icon :size="getIconSize()">mdi-code-tags</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="linkValue() ? 'tonal' : 'plain'" density="comfortable" :input-value="linkValue()" :disabled="linkButtonDisabled()" @click="linkClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_link')">
                  <v-icon :size="getIconSize()">mdi-link-variant</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="embedClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_embed')">
                <v-icon :size="getIconSize()">mdi-youtube</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="imageClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_image')">
                  <v-icon :size="getIconSize()">mdi-image-outline</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="videoClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_video')">
                  <v-icon :size="getIconSize()">mdi-video</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="audioClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_audio')">
                  <v-icon :size="getIconSize()">mdi-music</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="bulletListValue() ? 'tonal' : 'plain'" density="comfortable" @click="bulletListClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_bullet_list')">
                  <v-icon :size="getIconSize()">mdi-format-list-bulleted</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="orderedListValue() ? 'tonal' : 'plain'" density="comfortable" @click="orderedListClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_ordered_list')">
                  <v-icon :size="getIconSize()">mdi-format-list-numbered</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="textColorValue() ? 'tonal' : 'plain'" density="comfortable" @click="textColorClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_text_color')">
                  <v-icon :size="getIconSize()">mdi-invert-colors</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" :variant="backgroundColorValue() ? 'tonal' : 'plain'" density="comfortable" @click="backgroundColorClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_background_color')">
                  <v-icon :size="getIconSize()">mdi-format-color-fill</v-icon>
              </v-btn>

              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="smileyClick" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_smiley')">
                  <v-icon :size="getIconSize()">mdi-emoticon-outline</v-icon>
              </v-btn>

          </v-slide-group>

          <div class="custom-toolbar-send">
              <v-btn v-if="!this.editMessageDto.fileItemUuid" icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" :height="getBtnHeight()" @click="openFileUploadForAddingFiles()" :title="$vuetify.locale.t('$vuetify.message_edit_file')">
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
                      <v-btn icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" :height="getBtnHeight()" @click="onFilesClicked()" :title="$vuetify.locale.t('$vuetify.message_edit_attached_files')">
                        <v-icon :size="getIconSize()">mdi-file-document-multiple</v-icon>
                      </v-btn>
                  </v-badge>
              </template>
              <v-btn icon rounded="0" variant="plain" density="comfortable" :size="getBtnSize()" :width="getBtnWidth()" :height="getBtnHeight()" @click="resetInput()" :title="$vuetify.locale.t('$vuetify.message_edit_clear')">
                <v-icon :size="getIconSize()">mdi-delete</v-icon>
              </v-btn>
              <v-btn v-if="chatStore.canBroadcastTextMessage" icon rounded="0" :size="getBtnSize()" :variant="sendBroadcast ? 'tonal' : 'plain'" density="comfortable" @click="sendBroadcast = !sendBroadcast" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_broadcast')">
                  <v-icon :size="getIconSize()">mdi-broadcast</v-icon>
              </v-btn>
              <v-btn icon rounded="0" :size="getBtnSize()" variant="plain" density="comfortable" @click="openRecordingModal()" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.make_a_recording')">
                  <v-icon :size="getIconSize()">mdi-record-rec</v-icon>
              </v-btn>
              <v-btn icon rounded="0" variant="plain" :size="getBtnSize()" density="comfortable" @click="openMessageEditSettings()" :width="getBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_settings')">
                  <v-icon :size="getIconSize()">mdi-cog</v-icon>
              </v-btn>
              <v-btn color="primary" @click="sendMessageToChat" rounded="0" class="mr-0 ml-2 send" density="comfortable" icon="mdi-send" :width="sendMessageBtnWidth()" :height="getBtnHeight()" :title="$vuetify.locale.t('$vuetify.message_edit_send')" :disabled="sending" :loading="sending"></v-btn>
          </div>
        </div>
        <template v-else-if="shouldShowBubbleMenu">
            <bubble-menu
                class="bubble-menu"
                :tippy-options="{ duration: 0 }"
                :editor="this.$refs.tipTapRef.getEditor()"
            >
                <button @click="boldClick" :class="{ 'is-active': boldValue() }">
                    {{ $vuetify.locale.t('$vuetify.message_edit_bold_short') }}
                </button>
                <button @click="italicClick" :class="{ 'is-active': italicValue() }">
                    {{ $vuetify.locale.t('$vuetify.message_edit_italic_short') }}
                </button>
                <button @click="underlineClick" :class="{ 'is-active': underlineValue() }">
                    {{ $vuetify.locale.t('$vuetify.message_edit_underline_short') }}
                </button>
            </bubble-menu>

            <floating-menu
                class="floating-menu"
                :tippy-options="{ duration: 0, zIndex: 200, interactive: true, appendTo: documentBody }"
                :editor="this.$refs.tipTapRef.getEditor()"
            >
                <button @click="bulletListClick" :class="{ 'is-active': bulletListValue() }">
                    {{ $vuetify.locale.t('$vuetify.message_edit_bullet_list_short') }}
                </button>
                <button @click="orderedListClick" :class="{ 'is-active': orderedListValue() }">
                    {{ $vuetify.locale.t('$vuetify.message_edit_ordered_list_short') }}
                </button>

                <button @click="imageClick">
                    {{ $vuetify.locale.t('$vuetify.message_edit_image_short') }}
                </button>
                <button @click="videoClick">
                    {{ $vuetify.locale.t('$vuetify.message_edit_video_short') }}
                </button>
                <button @click="embedClick">
                    {{ $vuetify.locale.t('$vuetify.message_edit_embed_short') }}
                </button>

            </floating-menu>
        </template>
      </v-container>

      <!-- We store modals outside of container in order they not to contribute into the height (as it is done in App.vue) -->
      <MessageEditLinkModal/>
      <MessageEditMediaModal/>
      <RecordingModal/>
    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
        CLOSE_EDIT_MESSAGE,
        MESSAGE_EDIT_LOAD_FILES_COUNT,
        COLOR_SET,
        MESSAGE_EDIT_LINK_SET,
        OPEN_FILE_UPLOAD_MODAL,
        OPEN_CHOOSE_COLOR,
        OPEN_MESSAGE_EDIT_LINK,
        OPEN_MESSAGE_EDIT_MEDIA,
        OPEN_MESSAGE_EDIT_SMILEY,
        OPEN_VIEW_FILES_DIALOG,
        PROFILE_SET,
        SET_EDIT_MESSAGE,
        SET_EDIT_MESSAGE_MODAL,
        MESSAGE_EDIT_SET_FILE_ITEM_UUID,
        ON_WINDOW_RESIZED,
        OPEN_SETTINGS,
        ON_MESSAGE_EDIT_SEND_BUTTONS_TYPE_CHANGED,
        OPEN_RECORDING_MODAL, FILE_CREATED,
    } from "./bus/bus";
    import debounce from "lodash/debounce";
    import Tiptap from './TipTapEditor.vue'
    import {
      chatEditMessageDtoFactory,
      colorBackground,
      colorText,
      embed, getAnswerPreviewFields, getEmbed, setEmbed,
      hasLength, haveEmbed, isChatRoute,
      link_dialog_type_add_link_to_text,
      link_dialog_type_add_media_embed, media_audio,
      media_image,
      media_video, new_message, reply_message, isMobileWidth,
    } from "@/utils";
    import {
        getStoredChatEditMessageDto, getStoredChatEditMessageDtoOrNull, getStoredMessageEditSendButtonsType,
        removeStoredChatEditMessageDto,
        setStoredChatEditMessageDto
    } from "@/store/localStore"

    import MessageEditLinkModal from "@/MessageEditLinkModal";
    import MessageEditMediaModal from "@/MessageEditMediaModal";
    import {mapStores} from "pinia";
    import {fileUploadingSessionTypeMessageEdit, useChatStore} from "@/store/chatStore";
    import throttle from "lodash/throttle";
    import {
        BubbleMenu,
        FloatingMenu,
    } from '@tiptap/vue-3';
    import chroma from "chroma-js/chroma-light.js";
    import RecordingModal from "@/RecordingModal.vue";
    import {v4 as uuidv4} from "uuid";

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
                targetElement: null,
                resizeObserver: null,
            }
        },
        methods: {
            getContent(){
                return this.$refs.tipTapRef.getContent();
            },
            sendMessageToChat() {
                if (this.messageTextIsPresent(this.editMessageDto.text) || this.editMessageDto.fileItemUuid) {
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

              this.chatStore.isEditingBigText = false;
              this.$refs.tipTapRef.resetFileItemUuid();
            },
            messageTextIsPresent(text) {
                return text && text !== ""
            },
            loadFilesCount() {
                if (this.editMessageDto.fileItemUuid) {
                    return axios.post(`/api/storage/${this.chatId}/file/count`, {
                        fileItemUuid: this.editMessageDto.fileItemUuid
                    })
                        .then((response) => {
                            this.fileCount = response.data.count;
                            return response
                        });
                } else {
                  return Promise.resolve(false)
                }
            },
            loadFilesCountAndResetFileItemUuidIfNeed({chatId}) {
                if (chatId == this.chatId) {
                    this.loadFilesCount().then((resp) => {
                        if (resp) {
                            if (this.fileCount === 0) {
                                this.editMessageDto.fileItemUuid = null;
                                this.saveToStore();
                            }
                        }
                    })
                }
            },

            resetAnswer() {
                this.showAnswer = false;
                this.answerOnPreview = null;
                this.editMessageDto.embedMessage = null;
                this.saveToStore();
            },
            loadEmbedPreviewIfNeed(dto) {
                this.showAnswer = false;
                this.answerOnPreview = null;
                if (haveEmbed(dto)) {
                    const {embedPreviewText, embedPreviewOwner} = getAnswerPreviewFields(dto);
                    axios.put('/api/chat/public/preview-without-html', {text: embedPreviewText, login: embedPreviewOwner}).then(({data}) => {
                        this.showAnswer = true;
                        this.answerOnPreview = data.text;
                    })
                }
            },
            onSetMessageFromModal({dto, actionType}) {
              // in case actionType == new_message we any way should check for existing and for reply
              const mbExisting = getStoredChatEditMessageDtoOrNull(this.chatId);
              this.removeOwnerFromSavedMessageIfNeed(mbExisting);

              if (actionType == new_message) { // in case actionType == new_message we any way should check for existing and for reply
                if (mbExisting) {
                  this.editMessageDto = mbExisting;
                  this.afterSetMessage();
                } else {
                  this.editMessageDto = chatEditMessageDtoFactory();
                  this.afterSetMessage();
                }
              } else {
                if (mbExisting && mbExisting.id == dto.id) {
                  if (actionType == reply_message) {
                    setEmbed(mbExisting, getEmbed(dto))
                  }
                  this.editMessageDto = mbExisting;
                  this.afterSetMessage();
                } else {
                  this.editMessageDto = dto;
                  this.afterSetMessage();
                }
              }
              if (this.editMessageDto.id) {
                this.$emit("setEditingTitle");
              }
            },
            onSetMessage({dto, actionType}) {
              if (actionType == reply_message) {
                setEmbed(this.editMessageDto, getEmbed(dto));
              } else {
                this.editMessageDto = dto;
              }
              this.afterSetMessage();
            },
            afterSetMessage() {
              this.saveToStore();
              this.setContentToEditorAndLoad();
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
                this.editMessageDto.text = val;

                this.saveToStore();

                if (this.sendBroadcast) {
                    this.notifyAboutBroadcast(false, val);
                } else {
                    this.notifyAboutTyping(val);
                }
                this.emitExpandEventIfNeed();
            },
            getBtnWidth() {
                if (this.isMobile()) {
                    return '54px'
                } else {
                    return '2.4em'
                }
            },
            sendMessageBtnWidth() {
                if (this.isMobile()) {
                    return '72px'
                } else {
                    return '3.4em'
                }
            },
            getBtnHeight() {
                if (this.isMobile()) {
                    return '48px'
                } else {
                    return '2em'
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
                const fileItemUuid = this.editMessageDto.fileItemUuid;
                this.chatStore.correlationId = uuidv4();
                bus.emit(OPEN_FILE_UPLOAD_MODAL, {showFileInput: true, fileItemUuid: fileItemUuid, shouldSetFileUuidToMessage: true, messageIdToAttachFiles: this.editMessageDto.id, fileUploadingSessionType: fileUploadingSessionTypeMessageEdit, correlationId: this.chatStore.correlationId});
            },
            onFilesClicked() {
                this.chatStore.correlationId = uuidv4();
                bus.emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid: this.editMessageDto.fileItemUuid, messageEditing: true, messageIdToDetachFiles: this.editMessageDto.id, fileUploadingSessionType: fileUploadingSessionTypeMessageEdit, correlationId: this.chatStore.correlationId});
            },
            onFileItemUuid({fileItemUuid, chatId}) {
              if (chatId == this.chatId) {
                this.$refs.tipTapRef.setFileItemUuid(fileItemUuid);
                this.editMessageDto.fileItemUuid = fileItemUuid;
                this.saveToStore();
              }
            },
            boldValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('bold')
            },
            boldClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleBold().run()
            },
            italicValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('italic')
            },
            italicClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleItalic().run()
            },
            underlineValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('underline')
            },
            underlineClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleUnderline().run()
            },
            strikeValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('strike')
            },
            codeValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('code')
            },
            codeBlockValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('codeBlock')
            },
            strikeClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleStrike().run()
            },
            codeClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleCode().run()
            },
            codeBlockClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleCodeBlock().run()
            },
            linkClick() {
                const previousUrl = this.$refs.tipTapRef.$data.editor.getAttributes('link').href;
                bus.emit(OPEN_MESSAGE_EDIT_LINK, {dialogType: link_dialog_type_add_link_to_text, previousUrl});
            },
            bulletListClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleBulletList().run()
            },
            orderedListClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleOrderedList().run()
            },
            bulletListValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('bulletList')
            },
            orderedListValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('orderedList')
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
                return this.$refs.tipTapRef?.$data.editor.isActive('link')
            },
            linkButtonDisabled() {
                return this.$refs.tipTapRef?.$data.editor.view.state.selection.empty;
            },
            imageClick() {
                bus.emit(
                  OPEN_MESSAGE_EDIT_MEDIA,
                  {
                    chatId: this.chatId,
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
                    chatId: this.chatId,
                    type: media_video,
                    fromDiskCallback: () => this.$refs.tipTapRef.addVideo(),
                    setExistingMediaCallback: this.$refs.tipTapRef.setVideo
                  },
                );
            },
            audioClick() {
                bus.emit(
                    OPEN_MESSAGE_EDIT_MEDIA,
                    {
                        chatId: this.chatId,
                        type: media_audio,
                        fromDiskCallback: () => this.$refs.tipTapRef.addAudio(),
                        setExistingMediaCallback: this.$refs.tipTapRef.setAudio
                    },
                );
            },
            embedClick() {
                bus.emit(OPEN_MESSAGE_EDIT_LINK, {dialogType: link_dialog_type_add_media_embed, mediaType: embed});
            },
            convertColor(givenColor) {
                return hasLength(givenColor) ? chroma(givenColor).hex().toUpperCase() : null;
            },
            textColorClick(){
                const givenColor = this.$refs.tipTapRef?.$data.editor.getAttributes('textStyle')?.color;
                const hexColor = this.convertColor(givenColor);
                bus.emit(OPEN_CHOOSE_COLOR, {colorMode: colorText, color: hexColor});
            },
            textColorValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('textStyle')
            },
            backgroundColorClick() {
                const givenColor = this.$refs.tipTapRef?.$data.editor.getAttributes('highlight')?.color;
                const hexColor = this.convertColor(givenColor);
                bus.emit(OPEN_CHOOSE_COLOR, {colorMode: colorBackground, color: hexColor});
            },
            backgroundColorValue() {
                return this.$refs.tipTapRef?.$data.editor.isActive('highlight')
            },
            smileyClick() {
                bus.emit(OPEN_MESSAGE_EDIT_SMILEY,
                  {
                    addSmileyCallback: (text) => this.$refs.tipTapRef.addText(text),
                    title: this.$vuetify.locale.t('$vuetify.message_edit_smiley')
                  }
                );
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
            removeOwnerFromSavedMessageIfNeed(editMessageDto) {
              if (editMessageDto && editMessageDto.ownerId && editMessageDto.ownerId != this.chatStore.currentUser?.id) {
                console.log("Removing owner from saved message")
                editMessageDto.ownerId = null;
                editMessageDto.id = null;
              }
            },
            loadFromStore() {
                const editMessageDto = getStoredChatEditMessageDto(this.chatId, chatEditMessageDtoFactory());
                this.removeOwnerFromSavedMessageIfNeed(editMessageDto);
                this.editMessageDto = editMessageDto;

                this.setContentToEditorAndLoad();
            },
            setContentToEditorAndLoad() {
              this.loadEmbedPreviewIfNeed(this.editMessageDto);
              this.loadFilesCount();
              if (!this.$refs.tipTapRef.messageHasMeaningfulContent(this.editMessageDto.text)) {
                this.chatStore.isEditingBigText = false;
              }
              this.$nextTick(()=>{
                this.$refs.tipTapRef.setContent(this.editMessageDto.text);
                if (hasLength(this.editMessageDto.fileItemUuid)) {
                  this.$refs.tipTapRef.setFileItemUuid(this.editMessageDto.fileItemUuid);
                }
              }).then(()=>{
                this.$refs.tipTapRef.setCursorToEnd();
              });
              this.emitExpandEventIfNeed();
            },
            emitExpandEventIfNeed() {
              if (hasLength(this.editMessageDto?.text)) {
                const numRows = this.editMessageDto.text.split("<p>").length - 1;
                const numCodes = this.editMessageDto.text.split("<code>").length - 1;
                if (numRows > 1 || numCodes >= 1) {
                  this.chatStore.isEditingBigText = true;
                }
              }
            },
            saveToStore() {
                setStoredChatEditMessageDto(this.editMessageDto, this.chatId);
            },
            removeFromStore() {
                removeStoredChatEditMessageDto(this.chatId);
            },
            setShouldShowSendMessageButtons() {
                const type = getStoredMessageEditSendButtonsType('auto');
                switch (type) { // see MessageEditSettingsModalContent
                    case 'auto':
                        const width = this.targetElement?.offsetWidth;
                        this.chatStore.shouldShowSendMessageButtons = this.isMobile() ? true : !isMobileWidth(width);
                        break;
                    case 'full':
                        this.chatStore.shouldShowSendMessageButtons = true;
                        break;
                    case 'compact':
                        this.chatStore.shouldShowSendMessageButtons = false;
                        break;
                }
            },
            updateShouldShowSendMessageButtons() {
                this.setShouldShowSendMessageButtons();
            },
            reloadTipTap() {
                // reload
                this.$refs.tipTapRef.clearContent();
                this.$nextTick(() => {
                    this.loadFromStore();
                })
            },
            openMessageEditSettings() {
                bus.emit(OPEN_SETTINGS, 'message_edit_settings')
            },
            openRecordingModal() {
                bus.emit(OPEN_RECORDING_MODAL, {fileItemUuid: this.editMessageDto.fileItemUuid})
            },
            onFileCreatedEvent(dto) {
                if (hasLength(this.chatStore.correlationId) && this.chatStore.correlationId == dto?.fileInfoDto?.correlationId) {
                    if (this.chatStore.sendMessageAfterMediaInsert && this.chatStore.fileUploadingSessionType == fileUploadingSessionTypeMessageEdit) {
                        this.sendMessageToChat();
                        this.chatStore.resetSendMessageAfterMediaInsertRoutine();
                    }
                }
            },
            documentBody() {
              return document.body
            },
        },
        computed: {
          ...mapStores(useChatStore),
          shouldShowBubbleMenu() {
            return this.$refs.tipTapRef && this.$refs.tipTapRef.getEditor()
          },
        },
        mounted() {
            bus.on(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.on(SET_EDIT_MESSAGE_MODAL, this.onSetMessageFromModal);
            bus.on(MESSAGE_EDIT_SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.on(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.on(COLOR_SET, this.onColorSet);
            bus.on(PROFILE_SET, this.onProfileSet);
            bus.on(MESSAGE_EDIT_LOAD_FILES_COUNT, this.loadFilesCountAndResetFileItemUuidIfNeed);
            bus.on(FILE_CREATED, this.onFileCreatedEvent);

            this.targetElement = document.getElementById('sendButtonContainer')
            this.$nextTick(()=>{
                this.setShouldShowSendMessageButtons();
            })
            bus.on(ON_WINDOW_RESIZED, this.updateShouldShowSendMessageButtons);
            bus.on(ON_MESSAGE_EDIT_SEND_BUTTONS_TYPE_CHANGED, this.setShouldShowSendMessageButtons);

            this.resizeObserver = new ResizeObserver(this.updateShouldShowSendMessageButtons);
            this.resizeObserver.observe(this.targetElement);
        },
        beforeUnmount() {
            bus.off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.off(SET_EDIT_MESSAGE_MODAL, this.onSetMessageFromModal);
            bus.off(MESSAGE_EDIT_SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.off(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.off(COLOR_SET, this.onColorSet);
            bus.off(PROFILE_SET, this.onProfileSet);
            bus.off(MESSAGE_EDIT_LOAD_FILES_COUNT, this.loadFilesCountAndResetFileItemUuidIfNeed);
            bus.off(ON_WINDOW_RESIZED, this.updateShouldShowSendMessageButtons);
            bus.off(ON_MESSAGE_EDIT_SEND_BUTTONS_TYPE_CHANGED, this.setShouldShowSendMessageButtons);
            bus.off(FILE_CREATED, this.onFileCreatedEvent);

            this.resizeObserver.disconnect();
            this.resizeObserver = null;
            this.targetElement = null;
        },
        created(){
            this.notifyAboutTyping = throttle(this.notifyAboutTyping, 500);
            this.notifyAboutBroadcast = debounce(this.notifyAboutBroadcast, 100, {leading:true, trailing:true});
            this.updateShouldShowSendMessageButtons = debounce(this.updateShouldShowSendMessageButtons, 100);
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
                    this.reloadTipTap()
                },
            },
            '$route': {
                handler: async function (newValue, oldValue) {
                    if (isChatRoute(newValue)) {
                        if (newValue.params.id != oldValue.params.id) {
                            console.debug("Chat id has been changed", oldValue.params.id, "->", newValue.params.id);
                            if (hasLength(newValue.params.id)) {
                                this.$nextTick(() => {
                                    this.loadFromStore();
                                })
                            }
                        }
                    }
                }
            },
        },
        components: {
            Tiptap,
            MessageEditLinkModal,
            MessageEditMediaModal,
            BubbleMenu,
            FloatingMenu,
            RecordingModal,
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

.bubble-menu {
    display: flex;
    background-color: #0D0D0D;
    padding: 0.2rem;
    border-radius: 0.5rem;

    button {
        border: none;
        background: none;
        color: #FFF;
        font-size: 0.85rem;
        font-weight: 500;
        padding: 0 0.2rem;
        opacity: 0.6;

        &:hover,
        &.is-active {
            opacity: 1;
        }
    }
}

.floating-menu {
    display: flex;
    background-color: #f1f1f1;
    padding: 0.2rem;
    border-radius: 0.5rem;

    button {
        border: none;
        background: none;
        font-size: 0.85rem;
        font-weight: 500;
        padding: 0 0.2rem;
        opacity: 0.6;

        &:hover,
        &.is-active {
            opacity: 1;
        }
    }
}
</style>
