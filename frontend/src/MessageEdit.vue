<template>
    <v-container id="sendButtonContainer" class="py-0 px-1 pb-1 d-flex flex-column" fluid :style="{height: messageEditHeight}">
            <tiptap
                :key="editorKey"
                v-model="editMessageDto.text"
                ref="tipTapRef"
                @keyup.native.ctrl.enter="sendMessageToChat"
                @keyup.native.esc="resetInput"
            />

            <div id="custom-toolbar">
                <!--<div class="custom-toolbar-format" v-if="$refs.tipTapRef != null && $refs.tipTapRef.$data.editor != null">
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('bold'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleBold().run()">
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'bold' }"></font-awesome-icon>
                    </button>
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('italic'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleItalic().run()">
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'italic' }"></font-awesome-icon>
                    </button>
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('underline')
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleUnderline().run()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'underline' }"></font-awesome-icon>
                    </button>
                    <button
                        :class="{
                          'richText__menu-item': true,
                          active: $refs.tipTapRef.$data.editor.isActive('strike'),
                        }"
                        @click="$refs.tipTapRef.$data.editor.chain().focus().toggleStrike().run()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'strikethrough' }"></font-awesome-icon>
                    </button>
                    <select class="ql-color" v-if="false"></select>
                    <select class="ql-background" v-if="false"></select>
                    <button
                        :disabled="linkButtonDisabled()"
                        :class="{
                          'richText__menu-item': !linkButtonDisabled(),
                          'richText__menu-item-disabled': linkButtonDisabled(),
                          active: $refs.tipTapRef.$data.editor.isActive('link'),
                        }"
                            @click="setLink()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'link' }"></font-awesome-icon>
                    </button>
                    <button
                        class="richText__menu-item"
                        @click="$refs.tipTapRef.addImage()"
                    >
                        <font-awesome-icon :icon="{ prefix: 'fa', iconName: 'image' }"></font-awesome-icon>
                    </button>
                </div>-->
                <!--<div class="d-flex flex-nowrap flex-row">-->
                <div class="d-flex flex-wrap flex-row">
                    <div style="max-width: 100%">
                        <v-slide-group
                            multiple
                            show-arrows
                        >
                            <v-btn-toggle
                                v-model="pressedButtons"
                                dense
                                tile
                                borderless
                                multiple
                            >
                                <v-btn icon :input-value="boldValue()" @click="boldClick">
                                    <v-icon>mdi-format-bold</v-icon>
                                </v-btn>

                                <v-btn icon :input-value="italicValue()" @click="italicClick">
                                    <v-icon>mdi-format-italic</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-format-underline</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-format-strikethrough-variant</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-link-variant</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-image-outline</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-palette</v-icon>
                                </v-btn>

                                <v-btn icon>
                                    <v-icon>mdi-select-color</v-icon>
                                </v-btn>

                            </v-btn-toggle>
                        </v-slide-group>
                    </div>

                    <div class="flex-grow-1">
                        <div class="custom-toolbar-send">
                            <v-btn v-if="!this.editMessageDto.fileItemUuid" icon tile class="mr-4" @click="openFileUpload()"><v-icon color="primary">mdi-file-upload</v-icon></v-btn>
                            <template v-if="this.editMessageDto.fileItemUuid">
                                <v-badge
                                    :value="fileCount"
                                    :content="fileCount"
                                    color="green"
                                    overlap
                                    left
                                >
                                    <v-btn icon tile @click="onFilesClicked()"><v-icon>mdi-file-document-multiple</v-icon></v-btn>
                                </v-badge>
                            </template>
                            <v-btn icon tile class="mr-4" @click="resetInput()"><v-icon>mdi-delete</v-icon></v-btn>
                            <v-switch dense hide-details class="ma-0 mr-4" v-model="sendBroadcast"
                                :label="$vuetify.lang.t('$vuetify.message_broadcast')"
                            ></v-switch>
                            <v-btn color="primary" @click="sendMessageToChat" tile class="mr-0"><v-icon color="white">mdi-send</v-icon></v-btn>
                        </div>
                    </div>

                </div>
            </div>

    </v-container>
</template>

<script>
    import axios from "axios";
    import bus, {
        CLOSE_EDIT_MESSAGE, OPEN_EDIT_MESSAGE,
        OPEN_FILE_UPLOAD_MODAL,
        OPEN_VIEW_FILES_DIALOG,
        SET_EDIT_MESSAGE, SET_FILE_ITEM_UUID,
    } from "./bus";
    import debounce from "lodash/debounce";
    import {mapGetters} from "vuex";
    import {GET_SEARCH_STRING, GET_USER, SET_SEARCH_STRING} from "./store";
    import Tiptap from './TipTapEditor.vue'

    const dtoFactory = () => {
        return {
            id: null,
            text: "",
            fileItemUuid: null,
        }
    };

    export default {
        props:['chatId', 'canBroadcast', 'fullHeight'],
        data() {
            return {
                editorKey: +new Date(),
                editMessageDto: dtoFactory(),
                fileCount: null,
                sendBroadcast: false,
                pressedButtons: []
            }
        },
        methods: {
            sendMessageToChat() {
                if (this.messageTextIsPresent()) {
                    (this.editMessageDto.id ? axios.put(`/api/chat/`+this.chatId+'/message', this.editMessageDto) : axios.post(`/api/chat/`+this.chatId+'/message', this.editMessageDto)).then(response => {
                        this.resetInput();
                        bus.$emit(CLOSE_EDIT_MESSAGE);
                    })
                }
            },
            resetInput() {
              console.log("Resetting text input");
              this.editMessageDto = dtoFactory();
              this.fileCount = null;
            },
            messageTextIsPresent() {
                return this.editMessageDto.text && this.editMessageDto.text !== "" && this.editMessageDto.text !== '<p><br></p>'
            },
            onSetMessage(dto) {
                if (!dto) {
                    this.editMessageDto = dtoFactory()
                } else {
                    this.editMessageDto = dto;
                }
                this.editorKey++;
                if (this.editMessageDto.fileItemUuid) {
                    axios.get(`/api/storage/${this.chatId}/file/count/${this.editMessageDto.fileItemUuid}`)
                        .then((response) => {
                            this.onFileItemUuid({fileItemUuid: this.editMessageDto.fileItemUuid, count: response.data.count})
                        });
                }
            },
            notifyAboutBroadcast(clear) {
                if (clear) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: null});
                } else if (this.messageTextIsPresent()) {
                    axios.put(`/api/chat/`+this.chatId+'/broadcast', {text: this.editMessageDto.text});
                }
            },
            notifyAboutTyping() {
                if (this.messageTextIsPresent()) {
                    axios.put(`/api/chat/` + this.chatId + '/typing');
                }
            },
            sendNotification() {
                if (this.sendBroadcast) {
                    this.notifyAboutBroadcast();
                } else {
                    this.notifyAboutTyping();
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
            setLink() {
                const previousUrl = this.$refs.tipTapRef.$data.editor.getAttributes('link').href;
                const url = window.prompt('URL', previousUrl);
                if (url === null) {
                    return
                }

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
            linkButtonDisabled() {
                const empty = this.$refs.tipTapRef.$data.editor.view.state.selection.empty;
                const disabled = empty;
                // console.debug("linkButtonDisabled", disabled);
                return disabled;
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
        },
        computed: {
            ...mapGetters({currentUser: GET_USER}),
            messageEditHeight() {
                return this.fullHeight ? 'calc(100vh - 56px - 56px)' : '100%'
            }
        },
        mounted() {
            bus.$on(SET_EDIT_MESSAGE, this.onSetMessage);
            if (this.$vuetify.breakpoint.smAndUp) {
                bus.$on(OPEN_EDIT_MESSAGE, this.onSetMessage);
            }
            bus.$on(SET_FILE_ITEM_UUID, this.onFileItemUuid);
        },
        beforeDestroy() {
            bus.$off(SET_EDIT_MESSAGE, this.onSetMessage);
            if (this.$vuetify.breakpoint.smAndUp) {
                bus.$off(OPEN_EDIT_MESSAGE, this.onSetMessage);
            }
            bus.$off(SET_FILE_ITEM_UUID, this.onFileItemUuid);
        },
        created(){
            this.notifyAboutTyping = debounce(this.notifyAboutTyping, 500, {leading:true, trailing:false});
            this.notifyAboutBroadcast = debounce(this.notifyAboutBroadcast, 100, {leading:true, trailing:true});
        },
        watch: {
            'editMessageDto.text': {
                handler: function (newValue, oldValue) {
                    this.sendNotification();
                },
            },
            sendBroadcast: {
                handler: function (newValue, oldValue) {
                    if (!newValue) {
                        this.notifyAboutBroadcast(true);
                    } else {
                        this.notifyAboutBroadcast();
                    }
                }
            },
            '$vuetify.lang.current': {
                handler: function (newValue, oldValue) {
                    this.editorKey++;
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