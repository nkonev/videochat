<template>
    <v-container id="sendButtonContainer" class="py-0 px-1 pb-1 d-flex flex-column" fluid :style="{height: messageEditHeight}"
                 @keyup.ctrl.enter="sendMessageToChat"
                 @keyup.esc="resetInput"
    >
            <tiptap
                :key="editorKey"
                ref="tipTapRef"
                @input="sendNotification"
            />

            <div id="custom-toolbar">
                <div class="d-flex flex-wrap flex-row">
                    <div style="max-width: 100%" v-if="$refs.tipTapRef">
                        <v-slide-group
                            multiple
                            show-arrows
                        >
                            <v-btn icon tile :input-value="boldValue()" @click="boldClick" width="48px" :color="boldValue() ? 'black' : ''" :title="$vuetify.lang.t('$vuetify.message_edit_bold')">
                                <v-icon>mdi-format-bold</v-icon>
                            </v-btn>

                            <v-btn icon tile :input-value="italicValue()" @click="italicClick" width="48px" :color="italicValue() ? 'black' : ''" :title="$vuetify.lang.t('$vuetify.message_edit_italic')">
                                <v-icon>mdi-format-italic</v-icon>
                            </v-btn>

                            <v-btn icon tile :input-value="underlineValue()" @click="underlineClick" width="48px" :color="underlineValue() ? 'black' : ''" :title="$vuetify.lang.t('$vuetify.message_edit_underline')">
                                <v-icon>mdi-format-underline</v-icon>
                            </v-btn>

                            <v-btn icon tile :input-value="strikeValue()" @click="strikeClick" width="48px" :color="strikeValue() ? 'black' : ''" :title="$vuetify.lang.t('$vuetify.message_edit_strike')">
                                <v-icon>mdi-format-strikethrough-variant</v-icon>
                            </v-btn>

                            <v-btn icon tile :input-value="linkValue()" :disabled="linkButtonDisabled()" @click="linkClick" width="48px" :color="linkValue() ? 'black' : ''" :title="$vuetify.lang.t('$vuetify.message_edit_link')">
                                <v-icon>mdi-link-variant</v-icon>
                            </v-btn>

                            <v-btn icon tile @click="imageClick" width="48px" :title="$vuetify.lang.t('$vuetify.message_edit_image')">
                                <v-icon>mdi-image-outline</v-icon>
                            </v-btn>

                            <v-btn icon tile @click="textColorClick" width="48px" :title="$vuetify.lang.t('$vuetify.message_edit_text_color')">
                                <v-icon>mdi-palette</v-icon>
                            </v-btn>

                            <v-btn icon tile @click="backgroundColorClick" width="48px" :title="$vuetify.lang.t('$vuetify.message_edit_background_color')">
                                <v-icon>mdi-select-color</v-icon>
                            </v-btn>

                        </v-slide-group>
                    </div>

                    <div class="flex-grow-1">
                        <div class="custom-toolbar-send">
                            <v-btn v-if="!this.editMessageDto.fileItemUuid" icon tile width="48px" @click="openFileUpload()" :title="$vuetify.lang.t('$vuetify.message_edit_file')"><v-icon color="primary">mdi-file-upload</v-icon></v-btn>
                            <template v-if="this.editMessageDto.fileItemUuid">
                                <v-badge
                                    :value="fileCount"
                                    :content="fileCount"
                                    color="green"
                                    overlap
                                    left
                                >
                                    <v-btn icon tile width="48px" @click="onFilesClicked()" :title="$vuetify.lang.t('$vuetify.message_edit_attached_files')"><v-icon>mdi-file-document-multiple</v-icon></v-btn>
                                </v-badge>
                            </template>
                            <v-btn icon tile width="48px" class="mr-2" @click="resetInput()" :title="$vuetify.lang.t('$vuetify.message_edit_clear')"><v-icon>mdi-delete</v-icon></v-btn>
                            <v-switch v-if="canBroadcast" dense hide-details class="ma-0 mr-4" v-model="sendBroadcast"
                                :label="$vuetify.lang.t('$vuetify.message_broadcast')"
                            ></v-switch>
                            <v-btn color="primary" @click="sendMessageToChat" tile class="mr-0" :title="$vuetify.lang.t('$vuetify.message_edit_send')" :disabled="sending" :loading="sending"><v-icon color="white">mdi-send</v-icon></v-btn>
                        </div>
                    </div>

                </div>
            </div>

    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
        CLOSE_EDIT_MESSAGE, MESSAGE_EDIT_COLOR_SET, MESSAGE_EDIT_LINK_SET,
        OPEN_FILE_UPLOAD_MODAL, OPEN_MESSAGE_EDIT_COLOR, OPEN_MESSAGE_EDIT_LINK,
        OPEN_VIEW_FILES_DIALOG,
        SET_EDIT_MESSAGE, SET_FILE_ITEM_UUID,
    } from "./bus";
    import debounce from "lodash/debounce";
    import {mapGetters} from "vuex";
    import {GET_CAN_BROADCAST_TEXT_MESSAGE, GET_USER} from "./store";
    import Tiptap from './TipTapEditor.vue'
    import {colorBackground, colorText} from "@/utils";

    const dtoFactory = () => {
        return {
            id: null,
            text: "",
            fileItemUuid: null,
        }
    };

    export default {
        props:['chatId', 'fullHeight'],
        data() {
            return {
                editorKey: +new Date(),
                editMessageDto: dtoFactory(),
                fileCount: null,
                sendBroadcast: false,
                sending: false,
            }
        },
        methods: {
            getContent(){
                return this.$refs.tipTapRef.getContent();
            },
            sendMessageToChat() {
                this.editMessageDto.text = this.getContent();
                if (this.messageTextIsPresent(this.editMessageDto.text)) {
                    this.sending = true;
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto))
                        .then(response => {
                            this.resetInput();
                            bus.$emit(CLOSE_EDIT_MESSAGE);
                        }).finally(() => {
                        this.sending = false;
                    })
                }
            },
            resetInput() {
              console.log("Resetting text input");
              this.$refs.tipTapRef.clearContent();
              this.editMessageDto = dtoFactory();
              this.fileCount = null;
            },
            messageTextIsPresent(text) {
                return text && text !== ""
            },
            onSetMessage(dto) {
                if (!dto) {
                    this.editMessageDto = dtoFactory();
                } else {
                    this.editMessageDto = dto;
                }
                this.$refs.tipTapRef.setContent(this.editMessageDto.text);
                if (this.editMessageDto.fileItemUuid) {
                    axios.get(`/api/storage/${this.chatId}/file/count/${this.editMessageDto.fileItemUuid}`)
                        .then((response) => {
                            this.onFileItemUuid({fileItemUuid: this.editMessageDto.fileItemUuid, count: response.data.count})
                        });
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
            sendNotification(val) {
                if (this.sendBroadcast) {
                    this.notifyAboutBroadcast(false, val);
                } else {
                    this.notifyAboutTyping(val);
                }
            },

            openFileUpload() {
                bus.$emit(OPEN_FILE_UPLOAD_MODAL, this.editMessageDto.fileItemUuid, true, false);
            },
            onFilesClicked() {
                bus.$emit(OPEN_VIEW_FILES_DIALOG, {chatId: this.chatId, fileItemUuid: this.editMessageDto.fileItemUuid, messageEditing: true});
            },
            onFileItemUuid({fileItemUuid, count}) {
                this.editMessageDto.fileItemUuid = fileItemUuid;
                this.fileCount = count;
                if (this.fileCount === 0) {
                    this.editMessageDto.fileItemUuid = null;
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
            strikeClick() {
                this.$refs.tipTapRef.$data.editor.chain().focus().toggleStrike().run()
            },
            linkClick() {
                const previousUrl = this.$refs.tipTapRef.$data.editor.getAttributes('link').href;
                bus.$emit(OPEN_MESSAGE_EDIT_LINK, previousUrl);
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
                this.$refs.tipTapRef.addImage()
            },
            textColorClick(){
                bus.$emit(OPEN_MESSAGE_EDIT_COLOR, colorText);
            },
            backgroundColorClick() {
                bus.$emit(OPEN_MESSAGE_EDIT_COLOR, colorBackground);
            },
            onColorSet(color, colorMode) {
                console.debug("Setting color", color, colorMode);
                if (colorMode == colorText) {
                    if (color) {
                        this.$refs.tipTapRef.$data.editor.chain().focus().setColor(color.hex).run()
                    } else {
                        this.$refs.tipTapRef.$data.editor.chain().focus().unsetColor().run()
                    }
                } else if (colorMode == colorBackground) {
                    if (color) {
                        this.$refs.tipTapRef.$data.editor.chain().focus().setHighlight({ color: color.hex }).run()
                    } else {
                        this.$refs.tipTapRef.$data.editor.chain().focus().unsetHighlight().run()
                    }
                }
            }
        },
        computed: {
            ...mapGetters({
                currentUser: GET_USER,
                canBroadcast: GET_CAN_BROADCAST_TEXT_MESSAGE,
            }),
            messageEditHeight() {
                return this.fullHeight ? 'calc(100vh - 56px - 48px)' : '100%'
            }
        },
        mounted() {
            bus.$on(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$on(SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.$on(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.$on(MESSAGE_EDIT_COLOR_SET, this.onColorSet);
            this.resetInput();
        },
        beforeDestroy() {
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            bus.$off(SET_FILE_ITEM_UUID, this.onFileItemUuid);
            bus.$off(MESSAGE_EDIT_LINK_SET, this.onMessageLinkSet);
            bus.$off(MESSAGE_EDIT_COLOR_SET, this.onColorSet);
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
            '$vuetify.lang.current': {
                handler: function (newValue, oldValue) {
                    this.editorKey++;
                    this.resetInput();
                },
            },
        },
        components: {
            Tiptap
        }
    }
</script>

<style lang="stylus">
$mobileWidth = 800px
$borderColor = rgba(0, 0, 0, 0.2)

#sendButtonContainer {
    min-height 25%
}

#custom-toolbar {
    border-top-width: 0
    border-bottom-style dashed
    border-left-style dashed
    border-right-style dashed
    border-width 1px
    border-color: $borderColor
}
.richText {
    border-color: $borderColor
}
@media screen and (max-width: $mobileWidth) {
    #custom-toolbar {
        border-width: 0
    }
    //border-left-width: 0
    //border-right-width: 0
}

.custom-toolbar-format {
    display: inline-flex;
    flex-grow: 0;
}
.custom-toolbar-send {
    display: flex;
    flex-grow: 10
    justify-content flex-end
    align-items center
}
</style>